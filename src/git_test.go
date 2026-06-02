package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitRoot_ReturnsRepoRoot(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	got, err := GitRoot(repoDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolve symlinks so /tmp and /private/tmp (macOS) match.
	want, _ := filepath.EvalSymlinks(repoDir)
	got, _ = filepath.EvalSymlinks(got)

	if got != want {
		t.Errorf("GitRoot = %q, want %q", got, want)
	}
}

func TestGitRoot_WorksFromSubdirectory(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	sub := filepath.Join(repoDir, "a", "b")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := GitRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want, _ := filepath.EvalSymlinks(repoDir)
	got, _ = filepath.EvalSymlinks(got)

	if got != want {
		t.Errorf("GitRoot from subdir = %q, want %q", got, want)
	}
}

func TestGitRoot_ErrorOutsideRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir() // not a git repo

	_, err := GitRoot(dir)
	if err == nil {
		t.Error("expected error for non-repo directory, got nil")
	}
}

func TestChangedFiles_BetweenBranches(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	// Create a feature branch with a new file and a modified file.
	cmd := exec.Command("git", "checkout", "-b", "feature")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}

	commitFile(t, repoDir, "new.txt", "new content\n", "add new.txt")
	commitFile(t, repoDir, "hello.txt", "modified\n", "modify hello.txt")

	files, err := ChangedFiles(repoDir, "main", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("got %d changed files, want 2: %v", len(files), files)
	}

	got := map[string]bool{}
	for _, f := range files {
		got[f] = true
	}
	if !got["hello.txt"] || !got["new.txt"] {
		t.Errorf("changed files = %v, want hello.txt and new.txt", files)
	}
}

func TestChangedFiles_NoDifferences(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	// Comparing main to itself should yield no changes.
	files, err := ChangedFiles(repoDir, "main", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no changed files, got %v", files)
	}
}

func TestChangedFiles_WorkingTree(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	// Modify a file in the working tree without committing.
	if err := os.WriteFile(filepath.Join(repoDir, "hello.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := ChangedFiles(repoDir, "main", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != "hello.txt" {
		t.Errorf("got %v, want [hello.txt]", files)
	}
}

func TestFileAtRef_ReturnsContent(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	got, err := FileAtRef(repoDir, "main", "hello.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello\n" {
		t.Errorf("FileAtRef = %q, want %q", got, "hello\n")
	}
}

func TestFileAtRef_ReturnsEmptyForMissingFile(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	got, err := FileAtRef(repoDir, "main", "nonexistent.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}

func TestFileInWorkTree_ReturnsContent(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	got, err := FileInWorkTree(repoDir, "hello.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello\n" {
		t.Errorf("FileInWorkTree = %q, want %q", got, "hello\n")
	}
}

func TestFileInWorkTree_ReturnsEmptyForMissingFile(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)

	got, err := FileInWorkTree(repoDir, "nope.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFileInWorkTree_ErrorOnUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission checks don't apply when running as root")
	}
	requireGit(t)
	repoDir := initTestRepo(t)

	path := filepath.Join(repoDir, "secret.txt")
	if err := os.WriteFile(path, []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Remove read permission.
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	_, err := FileInWorkTree(repoDir, "secret.txt")
	if err == nil {
		t.Error("expected error for unreadable file, got nil")
	}
}
