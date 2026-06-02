package main

import (
	"os/exec"
	"testing"
)

func TestComputeDiffs_DetectsAddedFile(t *testing.T) {
	requireGit(t)
	requireDifft(t)

	repoDir := initTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "feat")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}
	commitFile(t, repoDir, "added.go", "package foo\n", "add file")

	diffs, err := ComputeDiffs(repoDir, "main", "feat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	if diffs[0].Path != "added.go" {
		t.Errorf("path = %q, want %q", diffs[0].Path, "added.go")
	}
	if diffs[0].Status != "added" {
		t.Errorf("status = %q, want %q", diffs[0].Status, "added")
	}
	if diffs[0].OldSrc != "" {
		t.Error("expected empty OldSrc for added file")
	}
	if diffs[0].NewSrc != "package foo\n" {
		t.Errorf("NewSrc = %q, want %q", diffs[0].NewSrc, "package foo\n")
	}
}

func TestComputeDiffs_DetectsChangedFile(t *testing.T) {
	requireGit(t)
	requireDifft(t)

	repoDir := initTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "feat")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}
	commitFile(t, repoDir, "hello.txt", "changed\n", "modify")

	diffs, err := ComputeDiffs(repoDir, "main", "feat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	if diffs[0].Status != "changed" {
		t.Errorf("status = %q, want %q", diffs[0].Status, "changed")
	}
	if diffs[0].OldSrc != "hello\n" {
		t.Errorf("OldSrc = %q, want %q", diffs[0].OldSrc, "hello\n")
	}
	if diffs[0].NewSrc != "changed\n" {
		t.Errorf("NewSrc = %q, want %q", diffs[0].NewSrc, "changed\n")
	}
}

func TestComputeDiffs_DetectsRemovedFile(t *testing.T) {
	requireGit(t)
	requireDifft(t)

	repoDir := initTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "feat")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}

	// Remove the file and commit.
	rm := exec.Command("git", "rm", "hello.txt")
	rm.Dir = repoDir
	if out, err := rm.CombinedOutput(); err != nil {
		t.Fatalf("git rm: %v\n%s", err, out)
	}
	ci := exec.Command("git", "commit", "-m", "remove hello.txt")
	ci.Dir = repoDir
	if out, err := ci.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}

	diffs, err := ComputeDiffs(repoDir, "main", "feat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	if diffs[0].Status != "removed" {
		t.Errorf("status = %q, want %q", diffs[0].Status, "removed")
	}
	if diffs[0].NewSrc != "" {
		t.Error("expected empty NewSrc for removed file")
	}
}

func TestComputeDiffs_NoDifferences(t *testing.T) {
	requireGit(t)
	requireDifft(t)

	repoDir := initTestRepo(t)

	diffs, err := ComputeDiffs(repoDir, "main", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected no diffs, got %d", len(diffs))
	}
}

func TestComputeDiffs_PopulatesLanguage(t *testing.T) {
	requireGit(t)
	requireDifft(t)

	repoDir := initTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "feat")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}
	commitFile(t, repoDir, "main.py", "print('hi')\n", "add python file")

	diffs, err := ComputeDiffs(repoDir, "main", "feat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	if diffs[0].Language == "" {
		t.Error("expected Language to be populated")
	}
}

func TestComputeDiffs_MultipleFiles(t *testing.T) {
	requireGit(t)
	requireDifft(t)

	repoDir := initTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "feat")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}
	commitFile(t, repoDir, "a.txt", "aaa\n", "add a")
	commitFile(t, repoDir, "b.txt", "bbb\n", "add b")
	commitFile(t, repoDir, "hello.txt", "changed\n", "modify hello")

	diffs, err := ComputeDiffs(repoDir, "main", "feat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diffs) != 3 {
		t.Errorf("got %d diffs, want 3", len(diffs))
	}

	paths := map[string]bool{}
	for _, d := range diffs {
		paths[d.Path] = true
	}
	for _, want := range []string{"a.txt", "b.txt", "hello.txt"} {
		if !paths[want] {
			t.Errorf("missing diff for %s", want)
		}
	}
}
