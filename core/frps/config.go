package frps

import (
	"fmt"
	"strings"
)

// Config holds frps server configuration.
type Config struct {
	BindAddr    string
	BindPort    int
	BindUDPPort int
	KCPBindPort int
	Token       string
	// Admin API
	AdminAddr string
	AdminPort int
	// TLS
	TLSMode     bool
	TLSCertFile string
	TLSKeyFile  string
	// Dashboard
	DashboardAddr string
	DashboardPort int
	DashboardUser string
	DashboardPwd  string
	// Logging
	LogFile  string
	LogLevel string
	LogMaxDays int
}

// BuildINI generates frps.ini content from Config.
func BuildINI(cfg *Config) string {
	var b strings.Builder
	b.WriteString("[common]\n")
	b.WriteString(fmt.Sprintf("bind_addr = %s\n", cfg.BindAddr))
	b.WriteString(fmt.Sprintf("bind_port = %d\n", cfg.BindPort))
	if cfg.BindUDPPort > 0 {
		b.WriteString(fmt.Sprintf("bind_udp_port = %d\n", cfg.BindUDPPort))
	}
	if cfg.KCPBindPort > 0 {
		b.WriteString(fmt.Sprintf("kcp_bind_port = %d\n", cfg.KCPBindPort))
	}
	if cfg.Token != "" {
		b.WriteString(fmt.Sprintf("token = %s\n", cfg.Token))
	}
	if cfg.AdminAddr != "" && cfg.AdminPort > 0 {
		b.WriteString(fmt.Sprintf("admin_addr = %s\n", cfg.AdminAddr))
		b.WriteString(fmt.Sprintf("admin_port = %d\n", cfg.AdminPort))
	}
	if cfg.TLSMode {
		b.WriteString("tls_only = true\n")
		if cfg.TLSCertFile != "" {
			b.WriteString(fmt.Sprintf("tls_cert_file = %s\n", cfg.TLSCertFile))
		}
		if cfg.TLSKeyFile != "" {
			b.WriteString(fmt.Sprintf("tls_key_file = %s\n", cfg.TLSKeyFile))
		}
	}
	if cfg.DashboardAddr != "" {
		b.WriteString(fmt.Sprintf("dashboard_addr = %s\n", cfg.DashboardAddr))
	}
	if cfg.DashboardPort > 0 {
		b.WriteString(fmt.Sprintf("dashboard_port = %d\n", cfg.DashboardPort))
	}
	if cfg.DashboardUser != "" {
		b.WriteString(fmt.Sprintf("dashboard_user = %s\n", cfg.DashboardUser))
	}
	if cfg.DashboardPwd != "" {
		b.WriteString(fmt.Sprintf("dashboard_pwd = %s\n", cfg.DashboardPwd))
	}
	if cfg.LogFile != "" {
		b.WriteString(fmt.Sprintf("log_file = %s\n", cfg.LogFile))
	}
	if cfg.LogLevel != "" {
		b.WriteString(fmt.Sprintf("log_level = %s\n", cfg.LogLevel))
	}
	if cfg.LogMaxDays > 0 {
		b.WriteString(fmt.Sprintf("log_max_days = %d\n", cfg.LogMaxDays))
	}
	return b.String()
}

// DefaultConfig returns a sensible default frps configuration.
func DefaultConfig() *Config {
	return &Config{
		BindAddr:    "0.0.0.0",
		BindPort:    7000,
		AdminAddr:   "127.0.0.1",
		AdminPort:   7500,
		LogLevel:    "info",
	}
}
