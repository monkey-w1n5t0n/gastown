# Decision 010: VCS Abstraction Interface

**Status:** Proposed
**Date:** 2026-01-05
**Context:** First-class Jujutsu (jj) support requires abstracting VCS operations
**Parent:** gm-bxx (Epic: First-class Jujutsu support for gastown)

## Decision

Create an `internal/vcs` package with a `VCS` interface that abstracts version control
operations. Implement `GitVCS` using existing `git.Git` and `JjVCS` using a new `jj.JJ`
wrapper. Use a factory function to select the VCS implementation based on rig config.

## Context

Gas Town currently uses `internal/git` directly throughout the codebase (~50 files).
Adding jj support requires either:
1. **Fork everywhere**: Duplicate all git logic with jj variants (maintenance nightmare)
2. **Abstraction layer**: Single interface, multiple implementations (clean)

We choose abstraction. The interface must cover all operations currently used by:
- Refinery Engineer (merge queue processing)
- Rig Manager (rig setup, worktree/workspace management)
- Crew Manager (worker clone management)
- CLI commands (done, sling, worktree, status, etc.)

## Interface Design

### Core Types

```go
package vcs

// VCSType identifies the version control system
type VCSType string

const (
    VCSGit VCSType = "git"
    VCSJj  VCSType = "jj"
)

// Status represents working directory state
type Status struct {
    Clean     bool
    Modified  []string
    Added     []string
    Deleted   []string
    Untracked []string
}

// Workspace represents a git worktree or jj workspace
type Workspace struct {
    Path   string
    Branch string // git: branch name, jj: bookmark name
    Commit string // git: SHA, jj: commit ID
}

// UncommittedWork contains info about uncommitted changes
type UncommittedWork struct {
    HasChanges      bool
    StashCount      int
    UnpushedCommits int
    ModifiedFiles   []string
    UntrackedFiles  []string
}

// Common errors
var (
    ErrNotARepo       = errors.New("not a repository")
    ErrMergeConflict  = errors.New("merge conflict")
    ErrRebaseConflict = errors.New("rebase conflict")
    ErrAuthFailure    = errors.New("authentication failed")
)
```

### VCS Interface

```go
// VCS abstracts version control operations across git and jj
type VCS interface {
    // Type returns the VCS type (git or jj)
    Type() VCSType

    // WorkDir returns the working directory
    WorkDir() string

    // === Repository Setup ===

    // Clone clones a repository to dest
    Clone(url, dest string) error

    // CloneBare creates a bare/colocated clone for workspace architecture
    CloneBare(url, dest string) error

    // === Branch/Bookmark Operations ===

    // CurrentBranch returns the current branch (git) or active bookmark (jj)
    CurrentBranch() (string, error)

    // DefaultBranch returns the default branch name (e.g., "main")
    DefaultBranch() string

    // Checkout switches to the given ref
    Checkout(ref string) error

    // CreateBranch creates a new branch/bookmark
    CreateBranch(name string) error

    // CreateBranchFrom creates a branch/bookmark from a specific ref
    CreateBranchFrom(name, ref string) error

    // DeleteBranch deletes a local branch/bookmark
    DeleteBranch(name string, force bool) error

    // ListBranches returns branches matching pattern
    ListBranches(pattern string) ([]string, error)

    // BranchExists checks if branch exists locally
    BranchExists(name string) (bool, error)

    // RemoteBranchExists checks if branch exists on remote
    RemoteBranchExists(remote, branch string) (bool, error)

    // ResetBranch force-updates a branch to point to a ref
    ResetBranch(name, ref string) error

    // === Remote Operations ===

    // Fetch fetches from remote
    Fetch(remote string) error

    // FetchBranch fetches a specific branch
    FetchBranch(remote, branch string) error

    // Pull pulls from remote branch
    Pull(remote, branch string) error

    // Push pushes to remote branch
    Push(remote, branch string, force bool) error

    // DeleteRemoteBranch deletes a branch on remote
    DeleteRemoteBranch(remote, branch string) error

    // RemoteURL returns the URL for a remote
    RemoteURL(remote string) (string, error)

    // === Staging & Commits ===

    // Add stages files (git) or is no-op (jj auto-tracks)
    Add(paths ...string) error

    // Commit creates a commit with the given message
    Commit(message string) error

    // CommitAll stages all and commits (git) or describes current (jj)
    CommitAll(message string) error

    // === Status ===

    // Status returns the working directory status
    Status() (*Status, error)

    // HasUncommittedChanges returns true if there are uncommitted changes
    HasUncommittedChanges() (bool, error)

    // CheckUncommittedWork performs comprehensive uncommitted work check
    CheckUncommittedWork() (*UncommittedWork, error)

    // === Merge & Rebase ===

    // Merge merges a branch into current
    Merge(branch string) error

    // MergeNoFF merges with --no-ff and custom message
    MergeNoFF(branch, message string) error

    // Rebase rebases current onto ref
    Rebase(onto string) error

    // AbortMerge aborts a merge in progress
    AbortMerge() error

    // AbortRebase aborts a rebase in progress
    AbortRebase() error

    // CheckConflicts tests if source can merge into target cleanly
    // Returns list of conflicting files (empty if clean)
    CheckConflicts(source, target string) ([]string, error)

    // === Workspaces (Worktrees/Workspaces) ===

    // WorkspaceAdd creates a new workspace with a new branch
    WorkspaceAdd(path, branch string) error

    // WorkspaceAddDetached creates a workspace at a detached ref
    WorkspaceAddDetached(path, ref string) error

    // WorkspaceAddExisting creates a workspace for existing branch
    WorkspaceAddExisting(path, branch string) error

    // WorkspaceAddExistingForce creates workspace even if branch checked out elsewhere
    WorkspaceAddExistingForce(path, branch string) error

    // WorkspaceRemove removes a workspace
    WorkspaceRemove(path string, force bool) error

    // WorkspacePrune removes entries for deleted paths
    WorkspacePrune() error

    // WorkspaceList returns all workspaces
    WorkspaceList() ([]Workspace, error)

    // === Comparison & History ===

    // Rev returns the commit hash for a ref
    Rev(ref string) (string, error)

    // IsAncestor checks if ancestor is an ancestor of descendant
    IsAncestor(ancestor, descendant string) (bool, error)

    // CommitsAhead returns commits on branch not on base
    CommitsAhead(base, branch string) (int, error)

    // BranchCreatedDate returns when a branch was created
    BranchCreatedDate(branch string) (string, error)

    // BranchPushedToRemote checks if branch is pushed
    BranchPushedToRemote(branch, remote string) (bool, int, error)

    // StashCount returns number of stashes (git) or 0 (jj has no stash)
    StashCount() (int, error)

    // UnpushedCommits returns number of unpushed commits
    UnpushedCommits() (int, error)
}
```

### Factory Function

```go
// New creates a VCS instance for the given directory
func New(dir string) (VCS, error) {
    // Check for jj first (colocated has both .jj and .git)
    if hasJjRepo(dir) {
        return NewJjVCS(dir)
    }
    if hasGitRepo(dir) {
        return NewGitVCS(dir)
    }
    return nil, ErrNotARepo
}

// NewFromConfig creates a VCS based on rig configuration
func NewFromConfig(dir string, vcsType VCSType) (VCS, error) {
    switch vcsType {
    case VCSJj:
        return NewJjVCS(dir)
    case VCSGit:
        return NewGitVCS(dir)
    default:
        return New(dir) // auto-detect
    }
}

// NewWithGitDir creates a VCS with explicit git dir (for bare repos)
func NewWithGitDir(gitDir, workDir string, vcsType VCSType) (VCS, error) {
    switch vcsType {
    case VCSJj:
        return NewJjVCSWithDir(gitDir, workDir)
    case VCSGit:
        return NewGitVCSWithDir(gitDir, workDir)
    default:
        return nil, fmt.Errorf("VCS type required for explicit git dir")
    }
}
```

## Semantic Differences

### Staging Area
- **Git**: Explicit `git add` required before commit
- **Jj**: Auto-tracks everything, no staging area
- **Abstraction**: `Add()` is no-op for jj; `Commit()` works identically

### Branches vs Bookmarks
- **Git**: Branches track commits, stay fixed
- **Jj**: Bookmarks follow commits through rebases automatically
- **Abstraction**: Use "branch" terminology; jj implementation uses `jj bookmark`

### Worktrees vs Workspaces
- **Git**: `git worktree add/remove`, shares `.git` dir
- **Jj**: `jj workspace add/remove`, shares `.jj` dir (colocated shares `.git` too)
- **Abstraction**: Use "workspace" terminology (more general)

### Conflict Handling
- **Git**: Conflicts block operations, require resolution before continuing
- **Jj**: Conflicts can be committed as conflict markers, auto-resolved on rebase
- **Abstraction**: Use git semantics (block on conflict); jj implementation
  can translate conflict-committed state to error for consistency

### Stash
- **Git**: Has stash mechanism (`git stash push/pop`)
- **Jj**: No stash; use `jj new` to create a new change
- **Abstraction**: `StashCount()` returns 0 for jj; no stash push/pop in interface

## Implementation Plan

### Phase 1: Interface Definition (gm-bxx.5 - this task)
- Define `internal/vcs/vcs.go` with interface and types
- No implementation yet

### Phase 2: Git Implementation (part of gm-bxx.3)
- Create `internal/vcs/git.go`
- Wrapper that delegates to `internal/git.Git`
- Minimal changes to existing git package

### Phase 3: Jj Implementation (gm-bxx.2)
- Create `internal/jj/jj.go` (new jj CLI wrapper)
- Create `internal/vcs/jj.go` (VCS interface implementation)

### Phase 4: Migration (gm-bxx.3 continued)
- Update callers to use `vcs.VCS` interface
- Add VCS type to rig configuration
- Gradual migration of 50+ files

## Testing Strategy

- Unit tests per implementation (git_test.go, jj_test.go)
- Interface compliance tests (run same tests against both implementations)
- Integration tests with real git/jj repos

## Consequences

**Positive:**
- Single interface for all VCS operations
- New VCS support requires only new implementation, not codebase changes
- Cleaner abstraction for managers/commands

**Negative:**
- Indirection layer adds small overhead
- Some VCS-specific features may not fit the interface
- Two codepaths to maintain (git + jj implementations)

**Mitigations:**
- Type assertion available for VCS-specific operations
- Comprehensive interface covers 95%+ of current usage
- Shared test suite ensures parity

## Related

- gm-bxx: Epic: First-class Jujutsu support for gastown
- gm-bxx.2: Create internal/jj package (jj CLI wrapper)
- gm-bxx.3: Implement VCS abstraction layer (uses this design)
- internal/git/git.go: Current git implementation (797 lines)
