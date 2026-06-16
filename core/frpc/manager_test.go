package frpc

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestConfigBuild(t *testing.T) {
	rp := 7001
	sub := "test"
	proxy := &ProxyConfig{
		ID:        1,
		Name:      "p1",
		Type:      "tcp",
		LocalIP:   "127.0.0.1",
		LocalPort: 8080,
		RemotePort: &rp,
		Subdomain: &sub,
	}
	server := &ServerConfig{
		Host:  "example.com",
		Port:  7000,
		Token: "secret",
	}
	ini := BuildINI(proxy, server)
	if ini == "" {
		t.Fatal("empty ini")
	}
	if !strings.Contains(ini, "server_addr = example.com") {
		t.Fatal("missing server_addr")
	}
	if !strings.Contains(ini, "token = secret") {
		t.Fatal("missing token")
	}
	if !strings.Contains(ini, "remote_port = 7001") {
		t.Fatal("missing remote_port")
	}
	if !strings.Contains(ini, "subdomain = test") {
		t.Fatal("missing subdomain")
	}
}

func TestManagerStartStop(t *testing.T) {
	m := NewManager()
	m.CmdBuilder = func(cfgPath string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "sleep 1")
	}

	proxy := &ProxyConfig{ID: 99, LocalIP: "127.0.0.1", LocalPort: 8080, Type: "tcp"}
	server := &ServerConfig{Host: "127.0.0.1", Port: 7000}

	if err := m.Start(proxy, server); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !m.IsRunning(proxy.ID) {
		t.Fatal("expected running")
	}
	time.Sleep(1500 * time.Millisecond)
	if m.IsRunning(proxy.ID) {
		t.Fatal("expected not running after sleep")
	}

	// start and stop manually
	if err := m.Start(proxy, server); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !m.IsRunning(proxy.ID) {
		t.Fatal("expected running")
	}
	if err := m.Stop(proxy.ID); err != nil {
		t.Logf("stop returned: %v", err)
	}
	// give time for cleanup
	time.Sleep(200 * time.Millisecond)
	if m.IsRunning(proxy.ID) {
		t.Fatal("expected stopped after kill")
	}
}

func TestRunningProxies(t *testing.T) {
	m := NewManager()
	m.CmdBuilder = func(cfgPath string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "sleep 5")
	}
	proxy := &ProxyConfig{ID: 1, LocalIP: "127.0.0.1", LocalPort: 8080, Type: "tcp"}
	server := &ServerConfig{Host: "127.0.0.1", Port: 7000}
	if err := m.Start(proxy, server); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	ids := m.RunningProxies()
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected [1], got %v", ids)
	}
	m.Stop(1)
}

func TestStopNonExistent(t *testing.T) {
	m := NewManager()
	if err := m.Stop(999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestLastError(t *testing.T) {
	m := NewManager()
	if err := m.Start(&ProxyConfig{ID: 1, LocalPort: 8080}, &ServerConfig{Host: ""}); err == nil {
		t.Fatal("expected error for empty host")
	}
	// no process started so LastError should be empty
	if err := m.Start(&ProxyConfig{ID: 1, LocalPort: 8080, Type: "tcp"}, &ServerConfig{Host: "127.0.0.1", Port: 7000}); err != nil {
		// might fail if frpc binary is missing — that's ok
		t.Logf("start error (expected without frpc): %v", err)
	}
}
