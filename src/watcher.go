package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Broadcaster manages SSE clients and sends them reload signals.
type Broadcaster struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[chan struct{}]struct{}),
	}
}

func (b *Broadcaster) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan struct{}) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

func (b *Broadcaster) Notify() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- struct{}{}:
		default:
			// already has a pending notification
		}
	}
}

// WatchRepo watches the git working tree for file changes and notifies
// the broadcaster after a debounce period. Returns the watcher so the
// caller can close it on shutdown.
func WatchRepo(repoDir string, broadcast *Broadcaster) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}

	// Walk the repo and add directories (fsnotify is not recursive by default)
	err = filepath.WalkDir(repoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()

		// Skip .git and common noise directories
		if name == ".git" || name == "node_modules" || name == ".gradle" ||
			name == "build" || name == ".idea" || name == "target" {
			return filepath.SkipDir
		}

		// Skip hidden dirs (other than repo root)
		if strings.HasPrefix(name, ".") && path != repoDir {
			return filepath.SkipDir
		}

		_ = watcher.Add(path)
		return nil
	})

	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("walk repo: %w", err)
	}

	go debounceLoop(watcher, broadcast)

	return watcher, nil
}

func debounceLoop(watcher *fsnotify.Watcher, broadcast *Broadcaster) {
	var timer *time.Timer
	debounce := 300 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Ignore .git internal changes
			if strings.Contains(event.Name, "/.git/") || strings.Contains(event.Name, "\\.git\\") {
				continue
			}

			// Reset debounce timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, func() {
				broadcast.Notify()
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}
