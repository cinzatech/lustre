package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// FileDiff holds everything the frontend needs for one file.
type FileDiff struct {
	Path     string          `json:"path"`
	Language string          `json:"language"`
	Status   string          `json:"status"` // "changed", "added", "removed"
	OldSrc   string          `json:"old_src"`
	NewSrc   string          `json:"new_src"`
	Diff     json.RawMessage `json:"diff"` // raw difftastic JSON
}

// ComputeDiffs runs difftastic on all changed files between base and head.
func ComputeDiffs(repoDir, base, head string) ([]FileDiff, error) {
	files, err := ChangedFiles(repoDir, base, head)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	results := make([]FileDiff, len(files))
	var wg sync.WaitGroup
	errs := make([]error, len(files))

	for i, f := range files {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			fd, err := diffOneFile(repoDir, base, head, path)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = fd
		}(i, f)
	}

	wg.Wait()

	// collect errors
	var errMsgs []string
	for _, e := range errs {
		if e != nil {
			errMsgs = append(errMsgs, e.Error())
		}
	}
	if len(errMsgs) > 0 {
		return results, fmt.Errorf("some diffs failed: %s", strings.Join(errMsgs, "; "))
	}

	// filter out zero-value results
	var filtered []FileDiff
	for _, r := range results {
		if r.Path != "" {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

func diffOneFile(repoDir, base, head, path string) (FileDiff, error) {
	// Get old content from base ref
	oldSrc, err := FileAtRef(repoDir, base, path)
	if err != nil {
		return FileDiff{}, fmt.Errorf("reading %s at %s: %w", path, base, err)
	}

	// Get new content from head ref or working tree
	var newSrc string
	if head == "" {
		newSrc, err = FileInWorkTree(repoDir, path)
	} else {
		newSrc, err = FileAtRef(repoDir, head, path)
	}
	if err != nil {
		return FileDiff{}, fmt.Errorf("reading %s at %s: %w", path, head, err)
	}

	// Determine status
	status := "changed"
	if oldSrc == "" {
		status = "added"
	} else if newSrc == "" {
		status = "removed"
	}

	// Write temp files for difftastic
	tmpDir, err := os.MkdirTemp("", "lustre-*")
	if err != nil {
		return FileDiff{}, err
	}
	defer os.RemoveAll(tmpDir)

	// Use the original filename so difftastic can detect the language
	ext := filepath.Ext(path)
	oldPath := filepath.Join(tmpDir, "old"+ext)
	newPath := filepath.Join(tmpDir, "new"+ext)

	if err := os.WriteFile(oldPath, []byte(oldSrc), 0644); err != nil {
		return FileDiff{}, err
	}
	if err := os.WriteFile(newPath, []byte(newSrc), 0644); err != nil {
		return FileDiff{}, err
	}

	// Run difftastic
	cmd := exec.Command("difft", "--display", "json", oldPath, newPath)
	cmd.Env = append(os.Environ(), "DFT_UNSTABLE=yes")
	out, _ := cmd.Output()
	// difft returns exit code 1 for "files differ" which is fine

	// Parse just enough to get language, or use raw output
	var parsed struct {
		Language string `json:"language"`
		Status   string `json:"status"`
	}
	if len(out) > 0 {
		_ = json.Unmarshal(out, &parsed)
	}

	lang := parsed.Language
	if lang == "" {
		lang = strings.TrimPrefix(ext, ".")
	}

	// For added/removed files, difftastic may not produce useful output
	// In that case, we'll let the frontend handle it
	var diffJSON json.RawMessage
	if len(out) > 0 {
		diffJSON = json.RawMessage(out)
	} else {
		diffJSON = json.RawMessage(`null`)
	}

	return FileDiff{
		Path:     path,
		Language: lang,
		Status:   status,
		OldSrc:   oldSrc,
		NewSrc:   newSrc,
		Diff:     diffJSON,
	}, nil
}
