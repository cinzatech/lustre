package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
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

	// Find a free port
	port, err := freePort()
	if err != nil {
		log.Fatalf("no free port: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	url := fmt.Sprintf("http://%s", addr)

	// Start filesystem watcher
	broadcast := NewBroadcaster()
	WatchRepo(repoDir, broadcast)

	// Open browser
	go func() {
		openBrowser(url)
	}()

	fmt.Printf("\n  ✦ lustre\n")
	fmt.Printf("  %s → %s\n", base, head)
	fmt.Printf("  %s\n\n", url)
	fmt.Printf("  Watching for changes. Ctrl+C to stop.\n\n")

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Printf("\n  Stopped.\n")
		os.Exit(0)
	}()

	// Start server (blocks)
	if err := StartServer(addr, repoDir, base, head, broadcast); err != nil {
		log.Fatal(err)
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
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
	_ = cmd.Start()
}
