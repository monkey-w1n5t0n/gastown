package vcs

import (
	"fmt"
	"os"
	"path/filepath"
)

// New creates a VCS instance for the given directory, auto-detecting the VCS type.
// It checks for jj first (colocated repos have both .jj and .git).
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

// NewFromConfig creates a VCS based on explicit VCS type configuration.
// If vcsType is empty, auto-detects.
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

// NewWithGitDir creates a VCS with explicit git/jj directory.
// This is used for bare repos where gitDir points to the .git directory
// and workDir may be empty or point to a worktree/workspace.
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

// DetectVCSType returns the VCS type for a directory.
func DetectVCSType(dir string) (VCSType, error) {
	if hasJjRepo(dir) {
		return VCSJj, nil
	}
	if hasGitRepo(dir) {
		return VCSGit, nil
	}
	return "", ErrNotARepo
}

// hasJjRepo checks if dir contains a jj repository.
func hasJjRepo(dir string) bool {
	jjPath := filepath.Join(dir, ".jj")
	info, err := os.Stat(jjPath)
	return err == nil && info.IsDir()
}

// hasGitRepo checks if dir contains a git repository.
func hasGitRepo(dir string) bool {
	// Check for .git directory
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err == nil && info.IsDir() {
		return true
	}
	// Also check for .git file (worktree)
	if err == nil && !info.IsDir() {
		return true
	}
	return false
}

// NewGitVCS creates a new Git VCS instance.
// TODO: Implement in git.go (gm-bxx.3)
func NewGitVCS(dir string) (VCS, error) {
	return nil, fmt.Errorf("GitVCS not yet implemented (see gm-bxx.3)")
}

// NewGitVCSWithDir creates a Git VCS with explicit git directory.
// TODO: Implement in git.go (gm-bxx.3)
func NewGitVCSWithDir(gitDir, workDir string) (VCS, error) {
	return nil, fmt.Errorf("GitVCS not yet implemented (see gm-bxx.3)")
}

// NewJjVCS creates a new Jj VCS instance.
// TODO: Implement in jj.go (gm-bxx.2, gm-bxx.3)
func NewJjVCS(dir string) (VCS, error) {
	return nil, fmt.Errorf("JjVCS not yet implemented (see gm-bxx.2)")
}

// NewJjVCSWithDir creates a Jj VCS with explicit jj directory.
// TODO: Implement in jj.go (gm-bxx.2, gm-bxx.3)
func NewJjVCSWithDir(jjDir, workDir string) (VCS, error) {
	return nil, fmt.Errorf("JjVCS not yet implemented (see gm-bxx.2)")
}
