package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

var Version = "dev"

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("lustre %s\n", Version)
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: lustre <branch> [base-branch]\n")
		fmt.Fprintf(os.Stderr, "\n  Compare <branch> against [base-branch] (default: main)\n")
		fmt.Fprintf(os.Stderr, "  Opens a diff viewer in your browser with live reload.\n\n")
		os.Exit(1)
	}

	head := os.Args[1]
	base := "main"
	if len(os.Args) >= 3 {
		base = os.Args[2]
	}

	// Ensure difft is available
	if _, err := exec.LookPath("difft"); err != nil {
		fmt.Fprintf(os.Stderr, "error: difft (difftastic) not found in PATH\n")
		fmt.Fprintf(os.Stderr, "Install: https://difftastic.wilfred.me.uk/\n")
		os.Exit(1)
	}

	// Find git root from current directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	repoDir, err := GitRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Bind a listener. Try port 5522 first for a stable URL across restarts,
	// then fall back to an OS-assigned port.
	ln, err := listenPreferred()
	if err != nil {
		log.Fatalf("no free port: %v", err)
	}
	url := fmt.Sprintf("http://%s", ln.Addr())

	// Start filesystem watcher
	broadcast := NewBroadcaster()
	watcher, err := WatchRepo(repoDir, broadcast)
	if err != nil {
		log.Fatalf("watcher: %v", err)
	}

	// Shut down cleanly on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Open browser
	go openBrowser(url)

	fmt.Printf("\n  ✦ lustre\n")
	fmt.Printf("  %s → %s\n", base, head)
	fmt.Printf("  %s\n\n", url)
	fmt.Printf("  Watching for changes. Ctrl+C to stop.\n\n")

	// Start server (blocks until context is cancelled)
	if err := StartServer(ctx, ln, repoDir, base, head, broadcast); err != nil {
		log.Fatal(err)
	}

	watcher.Close()
	fmt.Printf("\n  Stopped.\n")
}

// listenPreferred tries port 5522 for a stable URL across restarts, then
// falls back to an OS-assigned port.
func listenPreferred() (net.Listener, error) {
	if ln, err := net.Listen("tcp", "127.0.0.1:5522"); err == nil {
		return ln, nil
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	if err := cmd.Start(); err == nil {
		// Reap the child process so it doesn't become a zombie.
		go cmd.Wait()
	}
}
