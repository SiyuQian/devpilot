package auth

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type mutableTestService struct {
	name      string
	loginErr  error
	logoutErr error
	logins    int
	logouts   int
}

func (s *mutableTestService) Name() string { return s.name }
func (s *mutableTestService) Login() error {
	s.logins++
	return s.loginErr
}
func (s *mutableTestService) Logout() error {
	s.logouts++
	return s.logoutErr
}
func (s *mutableTestService) IsLoggedIn() bool { return true }

func TestAuthRegisterCommands(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	RegisterCommands(root)
	for _, name := range []string{"login", "logout", "status"} {
		cmd, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("Find(%s): %v", name, err)
		}
		if cmd.Name() != name {
			t.Fatalf("cmd = %q, want %q", cmd.Name(), name)
		}
	}
}

func TestAuthCommandHelpers(t *testing.T) {
	oldRegistry := registry
	registry = map[string]Service{}
	defer func() { registry = oldRegistry }()

	svc := &mutableTestService{name: "ok"}
	Register(svc)
	if code := runLogin("ok"); code != 0 || svc.logins != 1 {
		t.Fatalf("runLogin ok code=%d logins=%d", code, svc.logins)
	}
	if code := runLogout("ok"); code != 0 || svc.logouts != 1 {
		t.Fatalf("runLogout ok code=%d logouts=%d", code, svc.logouts)
	}
	if code := runLogin("missing"); code == 0 {
		t.Fatalf("runLogin missing code=0")
	}
	Register(&mutableTestService{name: "bad-login", loginErr: errors.New("bad")})
	if code := runLogin("bad-login"); code == 0 {
		t.Fatalf("runLogin bad code=0")
	}
	Register(&mutableTestService{name: "bad-logout", logoutErr: errors.New("bad")})
	if code := runLogout("bad-logout"); code == 0 {
		t.Fatalf("runLogout bad code=0")
	}
}

func TestRunStatus(t *testing.T) {
	dir := t.TempDir()
	restore := OverrideConfigDir(dir)
	defer restore()

	out := captureAuthStdout(t, runStatus)
	if !strings.Contains(out, "No services configured") {
		t.Fatalf("empty status output = %q", out)
	}
	if err := Save("trello", ServiceCredentials{"token": "t"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out = captureAuthStdout(t, runStatus)
	if !strings.Contains(out, "trello: logged in") {
		t.Fatalf("status output = %q", out)
	}
}

func captureAuthStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy: %v", err)
	}
	return buf.String()
}
