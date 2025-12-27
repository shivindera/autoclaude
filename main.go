package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/henryaj/autoclaude/internal/watcher"
)

var version = "dev"

const usage = `autoclaude - Automatically resume Claude Code sessions after rate limits

USAGE:
    autoclaude [OPTIONS]

DESCRIPTION:
    autoclaude monitors other tmux panes in the current window for Claude Code
    rate limit messages. When a limit is detected, it waits for the reset time,
    then automatically sends the resume command to continue the session.

    Run this in a separate tmux pane alongside your Claude Code sessions.
    It will monitor all other panes in the same window.

OPTIONS:
    -v              Enable verbose/debug logging
    -version        Show version information
    -test           Test mode: wait 10s then send resume sequence to another pane
    -force DELAY    Periodically send input to Claude Code panes at DELAY
                    interval (e.g., -force 30s, -force 1m). Use when Claude stops
                    working mid-task without hitting rate limits.
                    Always sends a leading Enter, then "continue" (or custom text
                    if -force-text is set), then a final Enter.
    -force-text STR Optional text to send instead of "continue" in force mode.
                    The sequence sent is: Enter, STR, Enter.
    -panes IDS      Comma-separated list of pane IDs to target (e.g., "%0,%2").
                    If not specified, targets all other panes in the window.

EXAMPLE:
    # Split your tmux window and run autoclaude in one pane
    tmux split-window -h
    autoclaude

    # With verbose logging
    autoclaude -v

    # Force continue every 30 seconds
    autoclaude -force 30s

    # Force with custom text every minute
    autoclaude -force 1m -force-text "please continue working"

    # Target specific panes only
    autoclaude -force 30s -panes "%1,%3"

HOW IT WORKS:
    1. Polls all tmux panes in the current window every 5 seconds
    2. Detects rate limit messages (e.g., "Usage limit reached")
    3. Parses the reset time from the message
    4. Waits until the limit resets, plus a random 5-10 second delay
    5. Sends Enter (to dismiss any selector menu) then "continue" + Enter

    With -force mode:
    - Detects Claude Code panes by their input prompt (────> pattern)
    - Sends "continue" at the specified interval to keep Claude working

REQUIREMENTS:
    - Must be run inside a tmux session
    - Claude Code sessions must be in other panes of the same window
`

func main() {
	var (
		verbose       bool
		showVersion   bool
		testMode      bool
		forceInterval time.Duration
		forceText     string
		targetPanes   string
	)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&testMode, "test", false, "Test mode: wait 10s then send resume sequence")
	flag.DurationVar(&forceInterval, "force", 0, "Force continue at interval (e.g., 30s, 1m)")
	flag.StringVar(&forceText, "force-text", "", "Custom text to send in force mode (default: continue)")
	flag.StringVar(&targetPanes, "panes", "", "Comma-separated pane IDs to target (e.g., %0,%2)")
	flag.Parse()

	if showVersion {
		fmt.Printf("autoclaude v%s\n", version)
		os.Exit(0)
	}

	w, err := watcher.New(verbose, testMode, forceInterval, forceText, targetPanes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure you're running this inside a tmux session.")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	if err := w.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
