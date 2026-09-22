// Integration tests run the built jym binary as a subprocess against a
// fake CLI and a stub TypeSafe endpoint. Tests that need a terminal
// (suggestions, prompt interaction) live in integration_unix_test.go;
// this file holds the helpers and the pipe-safe cases.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// fakeCLISrc is a tiny CLI with two levels of subcommands. It prints
// its argv so tests can verify exactly what ran.
const fakeCLISrc = `package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--help" {
		fmt.Println("Commands:\n  pr    Manage pull requests\n  repo  Manage repositories")
		return
	}
	if len(args) == 2 && args[0] == "pr" && args[1] == "--help" {
		fmt.Println("Commands:\n  view    View a pull request\n  list    List pull requests\n  status  Show status")
		return
	}
	if len(args) >= 1 && args[0] == "exit" {
		code := 0
		if len(args) > 1 {
			code, _ = strconv.Atoi(args[1])
		}
		fmt.Println("exiting with", code)
		os.Exit(code)
	}
	fmt.Println("ran:", strings.Join(args, " "))
}
`

var (
	binOnce sync.Once
	binDir  string
	binErr  error
)

// buildBinaries compiles jym and the fake CLI once per test run.
func buildBinaries(t *testing.T) (jym, fakecli string) {
	t.Helper()
	binOnce.Do(func() {
		binDir, binErr = os.MkdirTemp("", "jym-it-*")
		if binErr != nil {
			return
		}
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		if binErr = goBuild(filepath.Join(binDir, "jym"+ext), "."); binErr != nil {
			return
		}
		src := filepath.Join(binDir, "fakecli.go")
		if binErr = os.WriteFile(src, []byte(fakeCLISrc), 0o644); binErr != nil {
			return
		}
		binErr = goBuild(filepath.Join(binDir, "fakecli"+ext), src)
	})
	if binErr != nil {
		t.Fatalf("building test binaries: %v", binErr)
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return filepath.Join(binDir, "jym"+ext), filepath.Join(binDir, "fakecli"+ext)
}

func goBuild(out, src string) error {
	cmd := exec.Command("go", "build", "-o", out, src)
	if b, err := cmd.CombinedOutput(); err != nil {
		return &buildError{src: src, out: string(b)}
	}
	return nil
}

type buildError struct{ src, out string }

func (e *buildError) Error() string { return "go build " + e.src + ": " + e.out }

// containsFold reports whether s contains sub, ignoring case — Windows
// paths disagree on case depending on how they were resolved.
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// testEnv returns an isolated environment: private XDG dirs and the
// fake binary on PATH. A non-empty apiURL also wires up the stub
// endpoint and a test key.
func testEnv(t *testing.T, apiURL string) []string {
	t.Helper()
	_, fakecli := buildBinaries(t)
	tmp := t.TempDir()
	for _, sub := range []string{"cache", "config", "state"} {
		if err := os.MkdirAll(filepath.Join(tmp, sub), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	stateDir := filepath.Join(tmp, "state")
	env := []string{
		"PATH=" + filepath.Dir(fakecli) + string(os.PathListSeparator) + os.Getenv("PATH"),
		"XDG_CACHE_HOME=" + filepath.Join(tmp, "cache"),
		"XDG_CONFIG_HOME=" + filepath.Join(tmp, "config"),
		"XDG_STATE_HOME=" + stateDir,
		"JYM_CONFIG=" + filepath.Join(tmp, "config", "absent.toml"),
		"HOME=" + tmp,
	}
	if apiURL != "" {
		env = append(env, "JYM_API_ENDPOINT="+apiURL, "TYPESAFE_API_KEY=test-key")
	} else {
		// No key: pre-decline first-run setup so interactive tests
		// exercise the offline path instead of blocking on the prompt.
		if err := os.MkdirAll(filepath.Join(stateDir, "jym"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, "jym", "state.json"),
			[]byte(`{"setup_declined":true}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return env
}

// jevStub serves canned Jev answers and counts requests. These tests
// exec the jym binary, so the stub must be a real loopback server —
// httptest.NewTestServer's in-memory network cannot cross a process
// boundary.
type jevStub struct {
	srv    *httptest.Server
	calls  int
	answer map[string]any // decoded "answers" object, nil → canned view answer
}

func (s *jevStub) start(t *testing.T) string {
	t.Helper()
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls++
		answers := s.answer
		if answers == nil {
			answers = map[string]any{
				"intended": map[string]any{
					"type": "choice", "choice": "view", "confidence": 0.9,
					"probabilities": map[string]float64{"view": 0.9, "list": 0.05, "status": 0.03, "__none__": 0.02},
				},
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"model": "jev-1.13.0", "answers": answers})
	}))
	t.Cleanup(s.srv.Close)
	return s.srv.URL
}

// runJym executes the binary attached to pipes (non-TTY) and returns
// stdout, stderr and the exit code.
func runJym(t *testing.T, env []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	jym, _ := buildBinaries(t)
	cmd := exec.Command(jym, args...)
	cmd.Env = env
	var so, se strings.Builder
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err == nil {
		return so.String(), se.String(), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return so.String(), se.String(), ee.ExitCode()
	}
	t.Fatalf("running jym: %v", err)
	return "", "", -1
}

// Non-TTY is the strictest gate: jym must never intervene, so hint mode
// cannot even print a suggestion — the command runs untouched.
func TestNonTTYPassthrough(t *testing.T) {
	stub := &jevStub{}
	env := append(testEnv(t, stub.start(t)), "JYM_MODE=hint")
	stdout, stderr, code := runJym(t, env, "--", "fakecli", "pr", "shw", "42")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "ran: pr shw 42") {
		t.Fatalf("command did not run: %q", stdout)
	}
	if strings.Contains(stderr, "Did you mean") {
		t.Fatalf("jym intervened in a non-TTY context: %q", stderr)
	}
	if stub.calls != 0 {
		t.Fatalf("Jev was called in a non-TTY context")
	}
}

// Exit status of the target must propagate through jym unchanged.
func TestExitCodePreserved(t *testing.T) {
	env := testEnv(t, "")
	_, _, code := runJym(t, env, "--", "fakecli", "exit", "3")
	if code != 3 {
		t.Fatalf("exit code %d, want 3", code)
	}
}

// --refresh must not delete anything outside the cache directory.
// (Traversal payloads are covered by the cache unit test; this checks
// the flag path stays healthy.)
func TestRefreshKeepsOutsideFiles(t *testing.T) {
	env := testEnv(t, "")
	tmp := t.TempDir()
	victim := filepath.Join(tmp, "precious.txt")
	if err := os.WriteFile(victim, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i, e := range env {
		if strings.HasPrefix(e, "XDG_CACHE_HOME=") {
			env[i] = "XDG_CACHE_HOME=" + filepath.Join(tmp, "cache")
		}
	}
	runJym(t, env, "--refresh", "--", "fakecli", "--help")
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("--refresh deleted a file outside the cache: %v", err)
	}
}

// Generated bash/zsh functions route selected commands through jym. Debug
// output proves that fakecli did not run directly; the non-TTY gate then
// makes the wrapped execution deterministic and network-free.
func TestShellIntegrationRoutesThroughJym(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		shellPath, err := exec.LookPath(shell)
		if err != nil {
			t.Run(shell, func(t *testing.T) { t.Skipf("%s not installed", shell) })
			continue
		}
		t.Run(shell, func(t *testing.T) {
			jym, fakecli := buildBinaries(t)
			env := append(testEnv(t, ""), "JYM_DEBUG=1")
			flags := []string{"--noprofile", "--norc", "-c"}
			if shell == "zsh" {
				flags = []string{"-dfc"}
			}
			script := `eval "$("$1" --shell-integration "$2" fakecli)"; fakecli "two words"`
			args := append(flags, script, "jym-shell-test", jym, shell)
			cmd := exec.Command(shellPath, args...)
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("shell integration failed: %v\n%s", err, out)
			}
			// Windows resolves through PATHEXT, which reports .EXE in
			// uppercase, so compare the path case-insensitively.
			if !containsFold(string(out), "[jym] executable: "+fakecli) {
				t.Fatalf("fakecli did not route through jym:\n%s", out)
			}
			if !strings.Contains(string(out), "ran: two words") {
				t.Fatalf("arguments did not reach fakecli:\n%s", out)
			}
		})
	}
}

func TestAllShellIntegrationRoutesExternalCommand(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		shellPath, err := exec.LookPath(shell)
		if err != nil {
			t.Run(shell, func(t *testing.T) { t.Skipf("%s not installed", shell) })
			continue
		}
		t.Run(shell, func(t *testing.T) {
			jym, fakecli := buildBinaries(t)
			env := append(testEnv(t, ""), "JYM_DEBUG=1")
			flags := []string{"--noprofile", "--norc", "-c"}
			if shell == "zsh" {
				flags = []string{"-dfc"}
			}
			script := `eval "$("$1" --shell-integration "$2" --all)"; fakecli all-mode`
			args := append(flags, script, "jym-shell-test", jym, shell)
			cmd := exec.Command(shellPath, args...)
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("all integration failed: %v\n%s", err, out)
			}
			if !containsFold(string(out), "[jym] executable: "+fakecli) {
				t.Fatalf("fakecli did not route through --all integration:\n%s", out)
			}
		})
	}
}
