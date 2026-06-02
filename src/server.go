package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"
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

// templateData holds the values injected into the HTML template.
type templateData struct {
	Base    string
	Head    string
	Version string
}

func StartServer(ctx context.Context, ln net.Listener, repoDir, base, head string, broadcast *Broadcaster) error {
	mux := http.NewServeMux()

	// Parse template once at startup; html/template auto-escapes values.
	tmpl, err := template.ParseFS(templateFS, "template.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, templateData{
			Base:    base,
			Head:    head,
			Version: Version,
		})
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

	// Blob endpoint — serves raw file content for binary previews.
	// Usage: /api/blob?side=old&path=assets/logo.png
	mux.HandleFunc("/api/blob", func(w http.ResponseWriter, r *http.Request) {
		side := r.URL.Query().Get("side")
		filePath := r.URL.Query().Get("path")
		if filePath == "" || (side != "old" && side != "new") {
			http.Error(w, "missing side or path", 400)
			return
		}

		var data []byte
		var err error
		if side == "old" {
			data, err = RawFileAtRef(repoDir, base, filePath)
		} else if head == "" {
			data, err = RawFileInWorkTree(repoDir, filePath)
		} else {
			data, err = RawFileAtRef(repoDir, head, filePath)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if data == nil {
			http.NotFound(w, r)
			return
		}

		ct := detectMimeType(filePath, data)
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
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

		reqCtx := r.Context()
		for {
			select {
			case <-reqCtx.Done():
				return
			case <-ctx.Done():
				return
			case <-ch:
				fmt.Fprintf(w, "event: reload\ndata: changed\n\n")
				flusher.Flush()
			}
		}
	})

	srv := &http.Server{Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	log.Printf("listening on %s", ln.Addr())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
