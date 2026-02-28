package cli

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestParseArgsDefault(t *testing.T) {
	got := ParseArgs([]string{})
	if got != "config.yaml" {
		t.Fatalf("want config.yaml, got %s", got)
	}
}

func TestParseArgsShortFlag(t *testing.T) {
	got := ParseArgs([]string{"-c", "x.yaml"})
	if got != "x.yaml" {
		t.Fatalf("want x.yaml, got %s", got)
	}
}

func TestParseArgsLongFlag(t *testing.T) {
	got := ParseArgs([]string{"--config", "x.yaml"})
	if got != "x.yaml" {
		t.Fatalf("want x.yaml, got %s", got)
	}
}

func TestParseArgsMissingValueFallsBack(t *testing.T) {
	got := ParseArgs([]string{"--config"})
	if got != "config.yaml" {
		t.Fatalf("want config.yaml, got %s", got)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	_ = r.Close()
	return string(b)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	b, _ := io.ReadAll(r)
	_ = r.Close()
	return string(b)
}

func TestLoggerMethods(t *testing.T) {
	l := Logger{}
	stdout := captureStdout(t, func() {
		l.Info("i")
		l.Warn("w")
		l.Success("s")
		l.Failure("f")
	})
	stderr := captureStderr(t, func() {
		l.Error("e")
	})

	if !strings.Contains(stdout, "[INFO] i") {
		t.Fatalf("missing info output: %q", stdout)
	}
	if !strings.Contains(stdout, "[WARN] w") {
		t.Fatalf("missing warn output: %q", stdout)
	}
	if !strings.Contains(stdout, "[OK] s") {
		t.Fatalf("missing success output: %q", stdout)
	}
	if !strings.Contains(stdout, "[FAIL] f") {
		t.Fatalf("missing failure output: %q", stdout)
	}
	if !strings.Contains(stderr, "[ERROR] e") {
		t.Fatalf("missing error output: %q", stderr)
	}
}

func TestDownloadProgressUpdateAndStop(t *testing.T) {
	oldEnabled := defaultProgressManager.enabled
	defaultProgressManager.enabled = true
	defer func() { defaultProgressManager.enabled = oldEnabled }()

	p := NewDownloadProgress("seg")
	out := captureStdout(t, func() {
		p.Update(1, 0)
		p.Update(2, 5)
		p.Stop()
	})
	if !strings.Contains(out, "seg: 5/5") {
		t.Fatalf("missing progress output: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("stop should print newline: %q", out)
	}
}

func TestExit(t *testing.T) {
	if os.Getenv("RAJIDOU_TEST_EXIT") == "1" {
		Exit(7)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExit")
	cmd.Env = append(os.Environ(), "RAJIDOU_TEST_EXIT=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected process to exit with code")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("want exit code 7, got %d", exitErr.ExitCode())
	}
}
