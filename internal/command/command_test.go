package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCapturesOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := Runner{Stdout: &stdout, Stderr: &stderr}

	if err := runner.Run(context.Background(), Options{}, "sh", "-c", `printf 'ok'`); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if stdout.String() != "ok" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "ok")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPassesStdin(t *testing.T) {
	var stdout bytes.Buffer
	runner := Runner{
		Stdin:  strings.NewReader("pipeline payload"),
		Stdout: &stdout,
	}

	if err := runner.Run(context.Background(), Options{}, "sh", "-c", "cat"); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if stdout.String() != "pipeline payload" {
		t.Errorf("stdout = %q, want the stdin payload", stdout.String())
	}
}

func TestRunSetsWorkingDirectory(t *testing.T) {
	dir := t.TempDir()

	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout}

	if err := runner.Run(context.Background(), Options{Dir: dir}, "pwd"); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	want, absErr := filepath.Abs(dir)
	if absErr != nil {
		t.Fatalf("filepath.Abs: %v", absErr)
	}
	if got != want {
		t.Errorf("pwd = %q, want %q", got, want)
	}
}

func TestRunReportsExitStatus(t *testing.T) {
	runner := Runner{}

	err := runner.Run(context.Background(), Options{}, "sh", "-c", "exit 3")
	if err == nil {
		t.Fatal("Run() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "sh -c exit 3") || !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("Run() error = %q, want the wrapped exit status", err)
	}
	if _, ok := errors.AsType[*exec.ExitError](err); !ok {
		t.Errorf("Run() error = %q, want a wrapped exec.ExitError", err)
	}
}

func TestRunMissingCommand(t *testing.T) {
	if err := (Runner{}).Run(context.Background(), Options{}, "no-such-command-skm"); err == nil {
		t.Fatal("Run() succeeded, want error")
	} else if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("Run() error = %q, want exec.ErrNotFound", err)
	}
}

func TestRunEnvironmentOverride(t *testing.T) {
	var stdout bytes.Buffer
	runner := Runner{Stdout: &stdout}

	opts := Options{Env: append(os.Environ(), "SKM_TEST_VAR=injected")}
	if err := runner.Run(context.Background(), opts, "sh", "-c", `printf '%s' "$SKM_TEST_VAR"`); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if stdout.String() != "injected" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "injected")
	}
}

func TestRunContextCancelled(t *testing.T) {
	runner := Runner{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runner.Run(ctx, Options{}, "sh", "-c", "exit 0"); !errors.Is(err, context.Canceled) {
		t.Errorf("Run() error = %q, want a wrapped context.Canceled", err)
	}
}

func TestOutput(t *testing.T) {
	got, err := (Runner{}).Output(context.Background(), Options{}, "sh", "-c", `printf 'value'`)
	if err != nil {
		t.Fatalf("Output() error: %v", err)
	}
	if got != "value" {
		t.Errorf("Output() = %q, want %q", got, "value")
	}
}

func TestOutputIncludesStderrOnFailure(t *testing.T) {
	_, err := (Runner{}).Output(context.Background(), Options{}, "sh", "-c", `echo boom >&2; exit 4`)
	if err == nil {
		t.Fatal("Output() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Output() error = %q, want it to include stderr", err)
	}
}

func TestOutputMissingCommand(t *testing.T) {
	_, err := (Runner{}).Output(context.Background(), Options{}, "no-such-command-skm")
	if err == nil {
		t.Fatal("Output() succeeded, want error")
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("Output() error = %q, want exec.ErrNotFound", err)
	}
}

func TestCombined(t *testing.T) {
	got, err := (Runner{}).Combined(context.Background(), Options{}, "sh", "-c", `printf 'flow'`)
	if err != nil {
		t.Fatalf("Combined() error: %v", err)
	}
	if got != "flow" {
		t.Errorf("Combined() = %q, want %q", got, "flow")
	}
}

func TestCombinedReportsOutputOnFailure(t *testing.T) {
	_, err := (Runner{}).Combined(context.Background(), Options{}, "sh", "-c", `echo stage; echo boom >&2; exit 5`)
	if err == nil {
		t.Fatal("Combined() succeeded, want error")
	}

	wantParts := []string{"stage", "boom", "exit status 5"}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("Combined() error = %q, want it to include %q", err, part)
		}
	}
	if _, ok := errors.AsType[*exec.ExitError](err); !ok {
		t.Errorf("Combined() error = %q, want a wrapped exec.ExitError", err)
	}
}

func TestCombinedStdoutOnly(t *testing.T) {
	got, err := (Runner{}).Combined(context.Background(), Options{}, "sh", "-c", `echo only-stdout; exit 0`)
	if err != nil {
		t.Fatalf("Combined() error: %v", err)
	}
	if !strings.Contains(got, "only-stdout") {
		t.Errorf("Combined() = %q, want it to include the output", got)
	}
}

func TestRunnersWithoutStreamsWork(t *testing.T) {
	// Output/Combined must not panic with a zero-value Runner.
	if _, err := (Runner{}).Output(context.Background(), Options{}, "true"); err != nil {
		t.Fatalf("Output() error: %v", err)
	}
}
