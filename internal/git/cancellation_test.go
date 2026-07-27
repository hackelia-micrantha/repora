package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunReportsDeadlineExceededPromptly(t *testing.T) {
	binDir := t.TempDir()
	writeCancelableFakeGit(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldTimeout := gitTimeout
	gitTimeout = 25 * time.Millisecond
	t.Cleanup(func() { gitTimeout = oldTimeout })

	start := time.Now()
	err := run("", "sleep")
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
	if elapsed > time.Second {
		t.Fatalf("run took %s, want prompt timeout", elapsed)
	}
}

func TestRunContextReportsExplicitCancellationPromptly(t *testing.T) {
	binDir := t.TempDir()
	writeCancelableFakeGit(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldTimeout := gitTimeout
	gitTimeout = 5 * time.Second
	t.Cleanup(func() { gitTimeout = oldTimeout })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := runContext(ctx, "", "sleep")
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if elapsed > time.Second {
		t.Fatalf("runContext took %s, want prompt cancellation", elapsed)
	}
}

func TestRunCancellationTerminatesDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-tree termination is not guaranteed by the Windows command boundary")
	}

	binDir := t.TempDir()
	writeCancelableFakeGit(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldTimeout := gitTimeout
	gitTimeout = 25 * time.Millisecond
	t.Cleanup(func() { gitTimeout = oldTimeout })

	marker := filepath.Join(t.TempDir(), "descendant-survived")
	err := run("", "spawn-child", marker)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}

	time.Sleep(800 * time.Millisecond)
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("descendant process survived cancellation and wrote marker")
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("stat descendant marker: %v", statErr)
	}
}

func writeCancelableFakeGit(t *testing.T, binDir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		sourcePath := filepath.Join(binDir, "cancelable_fake_git.go")
		source := []byte(`package main

import (
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "spawn-child" {
		cmd := exec.Command(os.Args[0], "child", os.Args[2])
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Start()
		time.Sleep(3 * time.Second)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "child" {
		time.Sleep(500 * time.Millisecond)
		_ = os.WriteFile(os.Args[2], []byte("survived"), 0600)
		return
	}
	time.Sleep(3 * time.Second)
}
`)
		if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
			t.Fatalf("write fake git source: %v", err)
		}
		cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "git.exe"), sourcePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build fake git: %v: %s", err, out)
		}
		return
	}

	path := filepath.Join(binDir, "git")
	data := []byte(`#!/bin/sh
if [ "$1" = "spawn-child" ]; then
  marker="$2"
  (sleep 0.5; printf survived > "$marker") &
  wait
fi
sleep 3
`)
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
}
