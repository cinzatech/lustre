package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRoot returns the top-level directory of the git repository.
func GitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git not installed): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedFiles returns the list of files that differ between base and head.
// head can be a branch name or empty string for the working tree.
func ChangedFiles(repoDir, base, head string) ([]string, error) {
	var args []string
	if head == "" {
		// Compare base branch to working tree (staged + unstaged)
		args = []string{"diff", "--name-only", base}
	} else {
		args = []string{"diff", "--name-only", base + "..." + head}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only failed: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// FileAtRef returns the contents of a file at a given git ref.
// Returns empty string and no error if the file doesn't exist at that ref (new file).
func FileAtRef(repoDir, ref, path string) (string, error) {
	cmd := exec.Command("git", "show", ref+":"+path)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		// File doesn't exist at this ref — that's fine (added/deleted file)
		return "", nil
	}
	return string(out), nil
}

// FileInWorkTree returns the contents of a file in the working tree.
func FileInWorkTree(repoDir, path string) (string, error) {
	full := filepath.Join(repoDir, path)
	cmd := exec.Command("cat", full)
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	return string(out), nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
