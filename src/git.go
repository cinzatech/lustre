package main

import (
	"fmt"
	"os"
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

// RawFileAtRef returns the raw bytes of a file at a given git ref.
// Returns nil and no error if the file doesn't exist at that ref (new file).
func RawFileAtRef(repoDir, ref, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", ref+":"+path)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	return out, nil
}

// FileAtRef returns the contents of a file at a given git ref as a string.
// Returns empty string and no error if the file doesn't exist at that ref.
func FileAtRef(repoDir, ref, path string) (string, error) {
	data, err := RawFileAtRef(repoDir, ref, path)
	return string(data), err
}

// RawFileInWorkTree returns the raw bytes of a file in the working tree.
// Returns nil and no error if the file does not exist (deleted file).
func RawFileInWorkTree(repoDir, path string) ([]byte, error) {
	full := filepath.Join(repoDir, path)
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return data, nil
}

// FileInWorkTree returns the contents of a file in the working tree as a string.
// Returns empty string and no error if the file does not exist.
func FileInWorkTree(repoDir, path string) (string, error) {
	data, err := RawFileInWorkTree(repoDir, path)
	return string(data), err
}
