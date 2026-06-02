package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// startTestServer launches the real HTTP server on a free port and returns
// its base URL. The server runs in the background until the test ends.
func startTestServer(t *testing.T, repoDir, base, head string, b *Broadcaster) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	go func() {
		_ = StartServer(addr, repoDir, base, head, b)
	}()

	// Wait until the server is accepting connections.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			c.Close()
			return fmt.Sprintf("http://%s", addr)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("server did not start in time")
	return ""
}

func TestServer_RootReturnsHTMLWithInjectedValues(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()
	base := startTestServer(t, repoDir, "main", "feature/x", b)

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "main") {
		t.Error("response does not contain base branch name")
	}
	if !strings.Contains(html, "feature/x") {
		t.Error("response does not contain head branch name")
	}
}

func TestServer_NonRootPathReturns404(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()
	base := startTestServer(t, repoDir, "main", "feat", b)

	resp, err := http.Get(base + "/nonexistent")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestServer_StaticAssetsAreServed(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()
	base := startTestServer(t, repoDir, "main", "feat", b)

	resp, err := http.Get(base + "/static/style.css")
	if err != nil {
		t.Fatalf("GET /static/style.css: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("expected non-empty CSS body")
	}
}

func TestServer_APIDiffsReturnsJSON(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()
	base := startTestServer(t, repoDir, "main", "main", b)

	resp, err := http.Get(base + "/api/diffs")
	if err != nil {
		t.Fatalf("GET /api/diffs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var dr DiffsResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if dr.Base != "main" {
		t.Errorf("Base = %q, want %q", dr.Base, "main")
	}
	if dr.Head != "main" {
		t.Errorf("Head = %q, want %q", dr.Head, "main")
	}
	if dr.Files == nil {
		t.Error("Files should be non-nil (empty slice)")
	}
}

func TestServer_APIDiffsReturnsEmptyFilesWhenNoDiff(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()
	base := startTestServer(t, repoDir, "main", "main", b)

	resp, err := http.Get(base + "/api/diffs")
	if err != nil {
		t.Fatalf("GET /api/diffs: %v", err)
	}
	defer resp.Body.Close()

	var dr DiffsResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(dr.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(dr.Files))
	}
}

func TestServer_EventsStreamSendsConnectedAndReload(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()
	base := startTestServer(t, repoDir, "main", "main", b)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(base + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Read the initial "connected" event.
	var gotConnected bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "event: connected") {
			gotConnected = true
			break
		}
	}
	if !gotConnected {
		t.Fatal("did not receive 'connected' SSE event")
	}

	// Trigger a reload and read the event.
	b.Notify()

	var gotReload bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "event: reload") {
			gotReload = true
			break
		}
	}
	if !gotReload {
		t.Fatal("did not receive 'reload' SSE event after Notify")
	}
}
