package frps

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BindAddr != "0.0.0.0" {
		t.Fatalf("expected 0.0.0.0, got %s", cfg.BindAddr)
	}
	if cfg.BindPort != 7000 {
		t.Fatalf("expected 7000, got %d", cfg.BindPort)
	}
}

func TestBuildINI(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Token = "test-token"
	cfg.AdminPort = 7500
	cfg.DashboardPort = 7501

	ini := BuildINI(cfg)
	if !strings.Contains(ini, "bind_addr = 0.0.0.0") {
		t.Fatal("missing bind_addr")
	}
	if !strings.Contains(ini, "bind_port = 7000") {
		t.Fatal("missing bind_port")
	}
	if !strings.Contains(ini, "token = test-token") {
		t.Fatal("missing token")
	}
	if !strings.Contains(ini, "admin_port = 7500") {
		t.Fatal("missing admin_port")
	}
	if !strings.Contains(ini, "dashboard_port = 7501") {
		t.Fatal("missing dashboard_port")
	}
}

func TestBuildINITLS(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TLSMode = true
	cfg.TLSCertFile = "/etc/certs/cert.pem"
	cfg.TLSKeyFile = "/etc/certs/key.pem"

	ini := BuildINI(cfg)
	if !strings.Contains(ini, "tls_only = true") {
		t.Fatal("missing tls_only")
	}
	if !strings.Contains(ini, "tls_cert_file = /etc/certs/cert.pem") {
		t.Fatal("missing tls_cert_file")
	}
	if !strings.Contains(ini, "tls_key_file = /etc/certs/key.pem") {
		t.Fatal("missing tls_key_file")
	}
}

func TestAdminClientBaseURL(t *testing.T) {
	c := NewAdminClient("127.0.0.1", 7500)
	expected := "http://127.0.0.1:7500"
	if c.BaseURL != expected {
		t.Fatalf("expected %s, got %s", expected, c.BaseURL)
	}
}
