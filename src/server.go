package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

//go:embed template.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// DiffsResponse is the JSON payload for /api/diffs.
type DiffsResponse struct {
	Base  string     `json:"base"`
	Head  string     `json:"head"`
	Files []FileDiff `json:"files"`
	Error string     `json:"error,omitempty"`
}

func StartServer(addr, repoDir, base, head string, broadcast *Broadcaster) error {
	mux := http.NewServeMux()

	// Serve static assets (CSS, JS)
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static embed: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Serve the HTML template
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := templateFS.ReadFile("template.html")
		if err != nil {
			http.Error(w, "template not found", 500)
			return
		}

		// Inject branch names and version into the template
		replacer := strings.NewReplacer(
			"{{.Base}}", base,
			"{{.Head}}", head,
			"{{.Version}}", Version,
		)
		html := replacer.Replace(string(data))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	// Diffs API
	mux.HandleFunc("/api/diffs", func(w http.ResponseWriter, r *http.Request) {
		diffs, err := ComputeDiffs(repoDir, base, head)

		resp := DiffsResponse{
			Base:  base,
			Head:  head,
			Files: diffs,
		}
		if err != nil {
			resp.Error = err.Error()
		}
		if resp.Files == nil {
			resp.Files = []FileDiff{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// SSE endpoint for live reload
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := broadcast.Subscribe()
		defer broadcast.Unsubscribe(ch)

		// Send initial ping so the browser knows connection is live
		fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				fmt.Fprintf(w, "event: reload\ndata: changed\n\n")
				flusher.Flush()
			}
		}
	})

	log.Printf("listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}
