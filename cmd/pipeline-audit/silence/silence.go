//go:build !windows

/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infinilabs.com */

// Package silence redirects fd 1 to /dev/null before any framework
// package init gets to log: the lazy global.Env() setup ("configuration
// auto reload enabled", "watching config: …") prints to stdout during
// imports — before main() runs, unreachable from Go code. seelog writes
// also flush asynchronously, so the descriptor stays quiet until the
// report is about to print (End), after a forced Flush drained them.
//
// Import it FIRST in main.go — with no dependency between same-level
// imports, gc initializes them in declaration order. Set
// PIPELINE_AUDIT_VERBOSE=1 to keep every framework log.
package silence

import (
	"os"

	"golang.org/x/sys/unix"
)

var savedStdout int

func init() {
	if _, verbose := os.LookupEnv("PIPELINE_AUDIT_VERBOSE"); verbose {
		return
	}
	Begin()
}

// Begin points fd 1 at /dev/null (no-op when already quiet or on error).
func Begin() {
	if savedStdout > 0 {
		return
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer devnull.Close()
	saved, err := unix.Dup(1)
	if err != nil {
		return
	}
	savedStdout = saved
	_ = unix.Dup2(int(devnull.Fd()), 1)
}

// End restores the real stdout (no-op when never begun).
func End() {
	if savedStdout <= 0 {
		return
	}
	_ = unix.Dup2(savedStdout, 1)
	_ = unix.Close(savedStdout)
	savedStdout = 0
}
