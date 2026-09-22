//go:build unix

// Interactive integration tests: jym only intervenes when stderr is a
// terminal, so these run the binary under a real pty.
package main

import (
	"bytes"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// runJymPTY executes jym with stdin/stdout/stderr on one pty — like a
// real terminal session — and returns the combined output and exit
// code. When keyInput is non-empty it is sent once the one-key prompt
// appears.
func runJymPTY(t *testing.T, env []string, keyInput string, args ...string) (string, int) {
	t.Helper()
	jym, _ := buildBinaries(t)
	cmd := exec.Command(jym, args...)
	cmd.Env = env
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty start: %v", err)
	}
	defer ptmx.Close()

	var mu sync.Mutex
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		var tmp [4096]byte
		for {
			n, err := ptmx.Read(tmp[:])
			if n > 0 {
				mu.Lock()
				buf.Write(tmp[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	output := func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}

	if keyInput != "" {
		deadline := time.Now().Add(15 * time.Second)
		for !strings.Contains(output(), "[n] cancel") {
			if time.Now().After(deadline) {
				t.Fatalf("prompt never appeared; output: %q", output())
			}
			time.Sleep(20 * time.Millisecond)
		}
		if _, err := ptmx.Write([]byte(keyInput)); err != nil {
			t.Fatalf("writing key: %v", err)
		}
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	select {
	case err = <-waitCh:
	case <-time.After(20 * time.Second):
		// A prompt with no scripted answer would block forever.
		cmd.Process.Kill()
		t.Fatalf("jym timed out; output: %q", output())
	}
	// Drain any trailing output.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	if err == nil {
		return output(), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return output(), ee.ExitCode()
	}
	t.Fatalf("jym failed: %v (output: %q)", err, output())
	return "", -1
}

// A valid command under a terminal must exec directly: no Jev call, no
// suggestion, output and exit code preserved.
func TestValidCommandPassesThrough(t *testing.T) {
	stub := &jevStub{}
	env := testEnv(t, stub.start(t))
	out, code := runJymPTY(t, env, "", "--", "fakecli", "pr", "view", "42")
	if code != 0 {
		t.Fatalf("exit %d, output: %q", code, out)
	}
	if !strings.Contains(out, "ran: pr view 42") {
		t.Fatalf("unexpected output: %q", out)
	}
	if stub.calls != 0 {
		t.Fatalf("Jev was called %d times for a valid command", stub.calls)
	}
	if strings.Contains(out, "Did you mean") {
		t.Fatalf("suggestion shown for a valid command: %q", out)
	}
}

// Hint mode under a TTY prints the suggestion and still runs as typed.
func TestUnknownCommandSuggests(t *testing.T) {
	stub := &jevStub{}
	env := append(testEnv(t, stub.start(t)), "JYM_MODE=hint")
	out, code := runJymPTY(t, env, "", "--", "fakecli", "pr", "shw", "42")
	if code != 0 {
		t.Fatalf("exit %d, output: %q", code, out)
	}
	if !strings.Contains(out, "Did you mean") || !strings.Contains(out, "fakecli pr view 42") {
		t.Fatalf("suggestion missing corrected command line: %q", out)
	}
	if !strings.Contains(out, "ran: pr shw 42") {
		t.Fatalf("original command did not run as typed: %q", out)
	}
	if stub.calls != 1 {
		t.Fatalf("expected exactly one Jev call, got %d", stub.calls)
	}
}

// __none__ is a real answer: no suggestion and no offline fallback.
func TestNoneAnswerSuppressesFallback(t *testing.T) {
	stub := &jevStub{answer: map[string]any{
		"intended": map[string]any{
			"type": "choice", "choice": "__none__", "confidence": 0.95,
			"probabilities": map[string]float64{"view": 0.2, "list": 0.1, "status": 0.1, "__none__": 0.6},
		},
	}}
	env := append(testEnv(t, stub.start(t)), "JYM_MODE=hint")
	out, code := runJymPTY(t, env, "", "--", "fakecli", "pr", "shw")
	if code != 0 {
		t.Fatalf("exit %d: %q", code, out)
	}
	if strings.Contains(out, "Did you mean") {
		t.Fatalf("suggestion shown despite __none__: %q", out)
	}
	if !strings.Contains(out, "ran: pr shw") {
		t.Fatalf("command did not run: %q", out)
	}
}

// A malformed Jev answer is rejected at the boundary: the injected
// "evilcmd" never reaches exec, even in auto mode.
func TestMalformedAnswerNeverExecutes(t *testing.T) {
	stub := &jevStub{answer: map[string]any{
		"intended": map[string]any{
			"type": "choice", "choice": "evilcmd", "confidence": 0.99,
			"probabilities": map[string]float64{"evilcmd": 0.99},
		},
	}}
	env := append(testEnv(t, stub.start(t)), "JYM_MODE=auto")
	out, _ := runJymPTY(t, env, "", "--", "fakecli", "pr", "shw")
	if strings.Contains(out, "ran: pr evilcmd") {
		t.Fatalf("arbitrary API choice was executed: %q", out)
	}
	if !strings.Contains(out, "ran: pr shw") {
		t.Fatalf("original command did not run: %q", out)
	}
}

// No API key → offline edit-distance suggestion, marked offline.
func TestOfflineFallback(t *testing.T) {
	env := append(testEnv(t, ""), "JYM_MODE=hint")
	out, code := runJymPTY(t, env, "", "--", "fakecli", "pr", "vie")
	if code != 0 {
		t.Fatalf("exit %d: %q", code, out)
	}
	if !strings.Contains(out, "Did you mean") || !strings.Contains(out, "fakecli pr view") {
		t.Fatalf("no offline suggestion: %q", out)
	}
	if !strings.Contains(out, "offline matching") {
		t.Fatalf("missing offline marker: %q", out)
	}
	if !strings.Contains(out, "ran: pr vie") {
		t.Fatalf("original command did not run: %q", out)
	}
}

// Prompt + Enter runs the corrected command.
func TestPromptEnterRunsCorrected(t *testing.T) {
	stub := &jevStub{}
	env := testEnv(t, stub.start(t))
	out, code := runJymPTY(t, env, "\r", "--", "fakecli", "pr", "shw", "42")
	if code != 0 {
		t.Fatalf("exit %d: %q", code, out)
	}
	if !strings.Contains(out, "ran: pr view 42") {
		t.Fatalf("corrected command did not run: %q", out)
	}
	if !strings.Contains(out, " \r\nran: pr view 42") {
		t.Fatalf("corrected command output did not restart at column zero: %q", out)
	}
}

// Prompt + 'o' runs the command as typed, and because it exited 0 the
// token is learned: the same invocation later passes through silently.
func TestPromptRunAsTypedLearns(t *testing.T) {
	stub := &jevStub{}
	env := testEnv(t, stub.start(t))
	out, code := runJymPTY(t, env, "o", "--", "fakecli", "pr", "shw")
	if code != 0 {
		t.Fatalf("exit %d: %q", code, out)
	}
	if !strings.Contains(out, "ran: pr shw") {
		t.Fatalf("typed command did not run: %q", out)
	}

	out, code = runJymPTY(t, env, "", "--", "fakecli", "pr", "shw")
	if code != 0 {
		t.Fatalf("second run exit %d: %q", code, out)
	}
	if strings.Contains(out, "Did you mean") {
		t.Fatalf("learned command still prompted: %q", out)
	}
	if !strings.Contains(out, "ran: pr shw") {
		t.Fatalf("learned command did not run: %q", out)
	}
}

// Prompt + 'n' cancels with the documented 127 status and runs nothing.
func TestPromptCancel(t *testing.T) {
	stub := &jevStub{}
	env := testEnv(t, stub.start(t))
	out, code := runJymPTY(t, env, "n", "--", "fakecli", "pr", "shw")
	if code != 127 {
		t.Fatalf("cancel exit %d, want 127: %q", code, out)
	}
	if strings.Contains(out, "ran:") {
		t.Fatalf("command ran despite cancel: %q", out)
	}
}
