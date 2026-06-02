package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repository with an initial commit and
// returns its path. The repo is automatically cleaned up when the test ends.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@test")
	run("git", "config", "user.name", "test")

	// Initial commit with a file so we have something to diff against.
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	return dir
}

// commitFile writes a file, stages it, and commits in the given repo.
func commitFile(t *testing.T, repoDir, path, content, msg string) {
	t.Helper()
	full := filepath.Join(repoDir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("add", path)
	git("commit", "-m", msg)
}

// requireGit fails the test if git is not installed.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatal("git not found in PATH")
	}
}

// requireDifft fails the test if difft is not installed.
func requireDifft(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("difft"); err != nil {
		t.Fatal("difft not found in PATH")
	}
}

// commitBinaryFile writes raw bytes, stages, and commits in the given repo.
func commitBinaryFile(t *testing.T, repoDir, path string, content []byte, msg string) {
	t.Helper()
	full := filepath.Join(repoDir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, content, 0644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("add", path)
	git("commit", "-m", msg)
}
