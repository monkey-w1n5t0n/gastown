// Package vcs provides a version control system abstraction layer.
//
// This package defines a common interface for VCS operations, enabling
// Gas Town to work with both git and jujutsu (jj) repositories.
//
// Design: docs/hop/decisions/010-vcs-abstraction-interface.md
package vcs

import "errors"

// VCSType identifies the version control system.
type VCSType string

const (
	// VCSGit represents a git repository.
	VCSGit VCSType = "git"
	// VCSJj represents a jujutsu (jj) repository.
	VCSJj VCSType = "jj"
)

// Common errors returned by VCS operations.
var (
	ErrNotARepo       = errors.New("not a repository")
	ErrMergeConflict  = errors.New("merge conflict")
	ErrRebaseConflict = errors.New("rebase conflict")
	ErrAuthFailure    = errors.New("authentication failed")
)

// Status represents the working directory state.
type Status struct {
	Clean     bool
	Modified  []string
	Added     []string
	Deleted   []string
	Untracked []string
}

// Workspace represents a git worktree or jj workspace.
type Workspace struct {
	Path   string // Filesystem path to the workspace
	Branch string // Branch name (git) or bookmark name (jj)
	Commit string // Commit SHA (git) or commit ID (jj)
}

// UncommittedWork contains information about uncommitted changes.
type UncommittedWork struct {
	HasChanges      bool
	StashCount      int
	UnpushedCommits int
	ModifiedFiles   []string
	UntrackedFiles  []string
}

// Clean returns true if there is no uncommitted work.
func (u *UncommittedWork) Clean() bool {
	return !u.HasChanges && u.StashCount == 0 && u.UnpushedCommits == 0
}

// VCS abstracts version control operations across git and jj.
//
// This interface covers the operations needed by Gas Town's managers:
//   - Refinery Engineer (merge queue processing)
//   - Rig Manager (rig setup, workspace management)
//   - Crew Manager (worker clone management)
//   - CLI commands (done, sling, worktree, status, etc.)
type VCS interface {
	// Type returns the VCS type (git or jj).
	Type() VCSType

	// WorkDir returns the working directory.
	WorkDir() string

	// === Repository Setup ===

	// Clone clones a repository to dest.
	Clone(url, dest string) error

	// CloneBare creates a bare clone (git) or colocated clone (jj)
	// for shared workspace architecture.
	CloneBare(url, dest string) error

	// === Branch/Bookmark Operations ===

	// CurrentBranch returns the current branch (git) or active bookmark (jj).
	CurrentBranch() (string, error)

	// DefaultBranch returns the default branch name (e.g., "main").
	DefaultBranch() string

	// Checkout switches to the given ref.
	Checkout(ref string) error

	// CreateBranch creates a new branch/bookmark at HEAD.
	CreateBranch(name string) error

	// CreateBranchFrom creates a branch/bookmark from a specific ref.
	CreateBranchFrom(name, ref string) error

	// DeleteBranch deletes a local branch/bookmark.
	DeleteBranch(name string, force bool) error

	// ListBranches returns branches/bookmarks matching pattern.
	ListBranches(pattern string) ([]string, error)

	// BranchExists checks if branch/bookmark exists locally.
	BranchExists(name string) (bool, error)

	// RemoteBranchExists checks if branch exists on remote.
	RemoteBranchExists(remote, branch string) (bool, error)

	// ResetBranch force-updates a branch to point to a ref.
	ResetBranch(name, ref string) error

	// === Remote Operations ===

	// Fetch fetches from remote.
	Fetch(remote string) error

	// FetchBranch fetches a specific branch from remote.
	FetchBranch(remote, branch string) error

	// Pull pulls from remote branch (fetch + merge/rebase).
	Pull(remote, branch string) error

	// Push pushes to remote branch.
	Push(remote, branch string, force bool) error

	// DeleteRemoteBranch deletes a branch on remote.
	DeleteRemoteBranch(remote, branch string) error

	// RemoteURL returns the URL for a remote.
	RemoteURL(remote string) (string, error)

	// === Staging & Commits ===

	// Add stages files for commit.
	// For jj, this is a no-op (jj auto-tracks everything).
	Add(paths ...string) error

	// Commit creates a commit with the given message.
	Commit(message string) error

	// CommitAll stages all changes and commits.
	// For jj, this describes the current working copy change.
	CommitAll(message string) error

	// === Status ===

	// Status returns the working directory status.
	Status() (*Status, error)

	// HasUncommittedChanges returns true if there are uncommitted changes.
	HasUncommittedChanges() (bool, error)

	// CheckUncommittedWork performs comprehensive uncommitted work check.
	CheckUncommittedWork() (*UncommittedWork, error)

	// === Merge & Rebase ===

	// Merge merges a branch into current.
	Merge(branch string) error

	// MergeNoFF merges with --no-ff flag and custom message.
	MergeNoFF(branch, message string) error

	// Rebase rebases current branch onto ref.
	Rebase(onto string) error

	// AbortMerge aborts a merge in progress.
	AbortMerge() error

	// AbortRebase aborts a rebase in progress.
	AbortRebase() error

	// CheckConflicts performs a test merge to check if source can merge
	// into target cleanly. Returns list of conflicting files (empty if clean).
	// The merge is aborted after checking - no changes are made.
	CheckConflicts(source, target string) ([]string, error)

	// === Workspaces (git worktrees / jj workspaces) ===

	// WorkspaceAdd creates a new workspace with a new branch.
	WorkspaceAdd(path, branch string) error

	// WorkspaceAddDetached creates a workspace at a detached ref.
	WorkspaceAddDetached(path, ref string) error

	// WorkspaceAddExisting creates a workspace for an existing branch.
	WorkspaceAddExisting(path, branch string) error

	// WorkspaceAddExistingForce creates workspace even if branch is checked out elsewhere.
	WorkspaceAddExistingForce(path, branch string) error

	// WorkspaceRemove removes a workspace.
	WorkspaceRemove(path string, force bool) error

	// WorkspacePrune removes workspace entries for deleted paths.
	WorkspacePrune() error

	// WorkspaceList returns all workspaces in the repository.
	WorkspaceList() ([]Workspace, error)

	// === Comparison & History ===

	// Rev returns the commit hash/ID for a ref.
	Rev(ref string) (string, error)

	// IsAncestor checks if ancestor is an ancestor of descendant.
	IsAncestor(ancestor, descendant string) (bool, error)

	// CommitsAhead returns number of commits on branch not on base.
	CommitsAhead(base, branch string) (int, error)

	// BranchCreatedDate returns when a branch was created (YYYY-MM-DD).
	BranchCreatedDate(branch string) (string, error)

	// BranchPushedToRemote checks if branch is pushed to remote.
	// Returns (pushed, unpushedCount, error).
	BranchPushedToRemote(branch, remote string) (bool, int, error)

	// StashCount returns number of stashes.
	// For jj, returns 0 (jj has no stash concept).
	StashCount() (int, error)

	// UnpushedCommits returns number of commits not pushed to upstream.
	UnpushedCommits() (int, error)
}
