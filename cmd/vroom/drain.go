package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"
)

// downFile marks the instance as draining: while it exists, /health returns
// 502 so load balancers stop routing to it before the process shuts down.
var downFile = "/tmp/vroom.down"

// runDrain creates the down file and then waits for the given duration, so it
// can be used as a shell-free Kubernetes preStop hook. It returns the process
// exit code.
func runDrain(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: vroom drain <duration>, e.g. vroom drain 25s")
		return 1
	}
	wait, err := time.ParseDuration(args[0])
	if err != nil || wait < 0 {
		fmt.Fprintf(os.Stderr, "invalid drain duration %q, expected e.g. 25s\n", args[0])
		return 1
	}
	f, err := os.OpenFile(downFile, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drain failed: %v\n", err)
		return 1
	}
	f.Close()
	time.Sleep(wait)
	return 0
}

// clearDownFile removes a down file left over from a previous run of the
// container, e.g. after an in-place restart that kept /tmp, so a fresh
// process doesn't report itself unhealthy forever.
func clearDownFile() {
	if err := os.Remove(downFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("can't remove down file", "path", downFile, "err", err)
	}
}
