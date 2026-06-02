package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
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
	Binary   bool            `json:"binary"`
	MimeType string          `json:"mime_type,omitempty"`
	OldSrc   string          `json:"old_src"`
	NewSrc   string          `json:"new_src"`
	Diff     json.RawMessage `json:"diff"` // raw difftastic JSON
}

// isBinary reports whether data looks like binary content by scanning
// the first 8 KB for null bytes (the same heuristic git uses).
func isBinary(data []byte) bool {
	n := 8192
	if len(data) < n {
		n = len(data)
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// detectMimeType returns a MIME type for the given path and content.
func detectMimeType(path string, data []byte) string {
	ext := filepath.Ext(path)
	if mt := mime.TypeByExtension(ext); mt != "" {
		return mt
	}
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
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
	// Read raw bytes so we can detect binary content.
	oldRaw, err := RawFileAtRef(repoDir, base, path)
	if err != nil {
		return FileDiff{}, fmt.Errorf("reading %s at %s: %w", path, base, err)
	}

	var newRaw []byte
	if head == "" {
		newRaw, err = RawFileInWorkTree(repoDir, path)
	} else {
		newRaw, err = RawFileAtRef(repoDir, head, path)
	}
	if err != nil {
		return FileDiff{}, fmt.Errorf("reading %s at %s: %w", path, head, err)
	}

	// Determine status
	status := "changed"
	if len(oldRaw) == 0 {
		status = "added"
	} else if len(newRaw) == 0 {
		status = "removed"
	}

	// Binary check: if either side is binary, treat the whole file as binary.
	if isBinary(oldRaw) || isBinary(newRaw) {
		// Pick whichever side has content for MIME detection.
		sample := newRaw
		if len(sample) == 0 {
			sample = oldRaw
		}
		return FileDiff{
			Path:     path,
			Language: strings.TrimPrefix(filepath.Ext(path), "."),
			Status:   status,
			Binary:   true,
			MimeType: detectMimeType(path, sample),
		}, nil
	}

	oldSrc := string(oldRaw)
	newSrc := string(newRaw)

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

	if err := os.WriteFile(oldPath, oldRaw, 0644); err != nil {
		return FileDiff{}, err
	}
	if err := os.WriteFile(newPath, newRaw, 0644); err != nil {
		return FileDiff{}, err
	}

	// Run difftastic. Exit code 1 means "files differ" which is expected.
	// Other non-zero codes indicate real failures.
	cmd := exec.Command("difft", "--display", "json", oldPath, newPath)
	cmd.Env = append(os.Environ(), "DFT_UNSTABLE=yes")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// exit code 1 = files differ, that's fine
		} else {
			return FileDiff{}, fmt.Errorf("difft %s: %w", path, err)
		}
	}

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
