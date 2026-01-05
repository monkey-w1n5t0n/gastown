# Git to Jujutsu (jj) Migration Design

**Issue:** gm-bxx.1
**Author:** gastown/polecats/slit
**Date:** 2026-01-05
**Status:** Draft

## Executive Summary

This document designs the migration path for Gas Town rigs from git-only to git/jj hybrid VCS support. The key insight is that jj's colocated mode allows gradual, reversible migration where both VCS systems work side-by-side.

## Current Architecture

### Rig Structure
```
<rig>/
├── .repo.git/           # Bare git repo (shared object store)
├── refinery/rig/        # Git worktree from .repo.git
├── polecats/<name>/     # Git worktrees (timestamped branches)
├── mayor/rig/           # Separate git clone
└── crew/<name>/         # Separate git clones
```

### Key Git Operations Used
- `git clone --bare` - creates `.repo.git`
- `git worktree add -b <branch> <path>` - creates polecats/refinery
- `git worktree remove` - cleanup
- `git rebase`, `git merge` - refinery merge queue
- Branch naming: `polecat/<name>-<timestamp>`

## Jujutsu Concepts Mapping

| Git Concept | Jj Equivalent | Notes |
|------------|---------------|-------|
| Branch | Bookmark | Bookmarks are lighter weight |
| Worktree | Workspace | `jj workspace add/remove` |
| Bare repo | N/A | Jj doesn't support bare repos |
| `git clone` | `jj git clone` | Creates colocated by default |
| `git rebase` | `jj rebase` | Conflicts stored as commits |
| `git merge` | `jj merge` | Also supports `jj squash` |

## Migration Strategy

### Phase 1: Colocated Mode (Low Risk)

**Goal:** Enable jj in existing git repos without disrupting git workflows.

**For separate clones (mayor/rig, crew/*):**
```bash
cd <rig>/mayor/rig
jj git init --colocate
```

This creates `.jj/` alongside `.git/`. Both commands work:
- Existing git operations continue unchanged
- New jj commands are available
- jj auto-imports/exports changes to git

**Benefits:**
- Zero disruption to existing workflows
- Agents can incrementally adopt jj
- Easy rollback: `rm -rf .jj/`

### Phase 2: Workspace Migration (Medium Risk)

**Challenge:** Bare repos (`.repo.git`) don't work with jj directly.

**Solution:** Convert bare repo to colocated regular repo with default workspace.

```bash
# Step 1: Clone bare to regular repo with colocated jj
git clone .repo.git .repo
cd .repo
jj git init --colocate

# Step 2: Convert existing worktrees to jj workspaces
jj workspace add ../refinery/rig --name refinery
jj workspace add ../polecats/<name> --name polecat-<name>
```

**New Structure:**
```
<rig>/
├── .repo.git/           # Keep for reference/rollback (optional)
├── .repo/               # New: Regular repo with jj colocated
│   ├── .git/
│   └── .jj/
├── refinery/rig/        # Now: jj workspace
├── polecats/<name>/     # Now: jj workspaces
├── mayor/rig/           # Separate clone + jj colocated
└── crew/<name>/         # Separate clones + jj colocated
```

### Phase 3: Bookmark Migration

**Current:** Branches like `polecat/slit-mk0j3qth`

**Future:** Bookmarks (same naming, different semantics)

```bash
# Create bookmark (replaces git branch -b)
jj bookmark create polecat/slit-$(date +%s)

# Track remote bookmark
jj bookmark track main@origin
```

**Key differences:**
- Bookmarks are local by default (no upstream tracking needed)
- `jj git push --bookmark <name>` pushes specific bookmarks
- Bookmarks can point to any change, not just commits

## Active Polecat Handling

### Scenario: Migration with Active Polecats

**Problem:** Polecats may have uncommitted work during migration.

**Solution:** Multi-phase migration with safety gates.

```
┌─────────────────────────────────────────────────────────────┐
│                    MIGRATION PHASES                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. PRE-MIGRATION CHECK                                      │
│     ├── Check all polecats idle (no in_progress issues)     │
│     ├── Verify no uncommitted changes (git status clean)    │
│     └── Create backup of .repo.git                          │
│                                                              │
│  2. FREEZE DISPATCHING                                       │
│     ├── Set rig.migration_in_progress = true                │
│     └── Block gt sling and polecat creation                 │
│                                                              │
│  3. CONVERT REPO                                             │
│     ├── git clone .repo.git .repo                           │
│     ├── jj git init --colocate (in .repo)                   │
│     └── Verify jj operations work                           │
│                                                              │
│  4. CONVERT WORKSPACES                                       │
│     ├── For each worktree: jj workspace add                 │
│     ├── Verify workspace operations                         │
│     └── Remove old git worktree metadata                    │
│                                                              │
│  5. UPDATE CONFIG                                            │
│     ├── Set rig.vcs = "jj" in config.json                   │
│     └── Update rig manager to use jj                        │
│                                                              │
│  6. RESUME OPERATIONS                                        │
│     └── Clear migration_in_progress flag                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Handling In-Progress Work

If polecats have work during migration:

1. **Wait for completion:** Preferred. Let active work finish.
2. **Force migration:** Save branch refs, migrate, restore as bookmarks.
3. **Abort migration:** Roll back if any step fails.

```bash
# Save active branches before migration
git -C .repo.git for-each-ref --format='%(refname:short)' refs/heads/polecat/ > .polecat-branches

# After migration, create bookmarks from saved refs
while read branch; do
  jj bookmark create "$branch" -r "$(git rev-parse $branch)"
done < .polecat-branches
```

## Rollback Strategy

### Level 1: Undo Colocated Init (Trivial)
```bash
rm -rf .jj/
```
Git repo is unchanged.

### Level 2: Restore Bare Repo Architecture
```bash
# If .repo.git backup exists
rm -rf .repo
mv .repo.git.bak .repo.git

# Recreate worktrees from bare
git worktree add -b main refinery/rig
```

### Level 3: Full Rig Restore
```bash
# If rig snapshot exists
rm -rf <rig>
tar xzf <rig>.snapshot.tar.gz
```

### Rollback Decision Matrix

| Symptom | Action |
|---------|--------|
| Jj command fails | Fall back to git command |
| Workspace corrupt | Recreate workspace |
| .jj directory corrupt | Delete and re-init |
| All jj operations fail | Remove .jj, use git only |
| Data loss suspected | Restore from .repo.git backup |

## Testing Approach

### Unit Tests

1. **VCS abstraction tests** (`internal/vcs/*_test.go`)
   - Test GitVCS and JjVCS implement same interface
   - Test workspace add/remove operations
   - Test merge/rebase operations

2. **Migration function tests**
   - Test colocate init on existing repo
   - Test workspace conversion
   - Test rollback functions

### Integration Tests

1. **Polecat lifecycle with jj**
   ```bash
   # Create polecat using jj workspace
   gt polecat add test-jj

   # Verify workspace created
   jj workspace list | grep test-jj

   # Sling work, complete, cleanup
   gt sling <issue> gastown --polecat test-jj
   gt done  # in polecat workspace
   gt polecat rm test-jj
   ```

2. **Refinery merge queue with jj**
   ```bash
   # Process merge request using jj rebase
   jj rebase -d main
   jj git push
   ```

3. **Mixed git/jj operations**
   ```bash
   # Ensure git commands still work in colocated repo
   git status
   git log
   git push origin HEAD
   ```

### Migration Test Script

```bash
#!/bin/bash
# test-jj-migration.sh

set -e

# Create test rig
gt rig add test-rig https://github.com/test/repo

# Run pre-migration checks
gt doctor test-rig

# Perform migration
gt rig migrate test-rig --vcs jj

# Verify jj operations
cd ~/gt/test-rig
jj status
jj log

# Test polecat lifecycle
gt polecat add test-cat
gt polecat rm test-cat

# Test refinery
gt mq test test-rig

# Clean up
gt rig rm test-rig --force
```

## Implementation Tasks

See parent epic gm-bxx for full task breakdown. Key milestones:

1. **gm-bxx.2**: Implement Go jj library wrapper
2. **gm-bxx.3**: VCS abstraction layer
3. **gm-bxx.6-9**: Manager refactoring (polecat, crew, refinery, witness)
4. **gm-bxx.10**: Merge queue redesign for jj conflicts
5. **gm-bxx.17-20**: Comprehensive testing

## Open Questions

1. **Should we support jj-only mode or always colocated?**
   - Recommendation: Always colocated for compatibility

2. **How to handle jj's conflict-as-commit model in merge queue?**
   - Refinery needs to detect conflict commits and handle appropriately

3. **Should mayor use jj workspaces or remain separate clone?**
   - Recommendation: Separate clone for isolation (mayor doesn't need branch visibility)

4. **What happens to existing polecat branches after migration?**
   - They become bookmarks automatically in colocated mode

## Appendix: Command Reference

### Jj Equivalents for Common Git Operations

```bash
# Clone
git clone <url>           →  jj git clone <url>

# Status
git status                →  jj status

# Add/Commit
git add . && git commit   →  jj commit  (auto-adds all changes)

# Branch
git checkout -b foo       →  jj bookmark create foo

# Rebase
git rebase main           →  jj rebase -d main

# Push
git push origin foo       →  jj git push --bookmark foo

# Fetch
git fetch origin          →  jj git fetch

# Worktree
git worktree add path     →  jj workspace add path
```

### Jj-Specific Useful Commands

```bash
# See all changes (like git log but better)
jj log

# Move between changes
jj edit <change-id>

# Squash recent changes
jj squash

# Split a change
jj split

# Describe a change (set commit message)
jj describe -m "message"

# Resolve conflicts (shows conflict markers in-place)
jj resolve
```

## References

- [Jujutsu Git Compatibility](http://docs.jj-vcs.dev/latest/git-compatibility/)
- [Working with GitHub](https://jj-vcs.github.io/jj/latest/github/)
- [Using Jujutsu in colocated repo](https://cuffaro.com/2025-03-15-using-jujutsu-in-a-colocated-git-repository/)
