package frps

import (
	"fmt"
	"os"
	"os/exec"
)

// Server manages an frps process lifecycle.
type Server struct {
	Config    *Config
	cmd       *exec.Cmd
	cfgPath   string
	CmdBuilder func(cfgPath string) *exec.Cmd
}

// NewServer creates a new frps server manager with the given config.
func NewServer(cfg *Config) *Server {
	return &Server{
		Config: cfg,
		CmdBuilder: func(cfgPath string) *exec.Cmd {
			return exec.Command("frps", "-c", cfgPath)
		},
	}
}

// Start launches the frps process.
func (s *Server) Start() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return fmt.Errorf("frps already running")
	}
	ini := BuildINI(s.Config)
	f, err := os.CreateTemp("", "asyou-frps-*.ini")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	s.cfgPath = f.Name()
	if _, err := f.WriteString(ini); err != nil {
		f.Close()
		os.Remove(s.cfgPath)
		return fmt.Errorf("write config: %w", err)
	}
	f.Close()

	cmd := s.CmdBuilder(s.cfgPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		os.Remove(s.cfgPath)
		return fmt.Errorf("start frps: %w", err)
	}
	s.cmd = cmd
	return nil
}

// Stop terminates the frps process.
func (s *Server) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	err := s.cmd.Process.Kill()
	if s.cfgPath != "" {
		os.Remove(s.cfgPath)
	}
	s.cmd = nil
	s.cfgPath = ""
	return err
}

// IsRunning checks if the frps process is active.
func (s *Server) IsRunning() bool {
	return s.cmd != nil && s.cmd.Process != nil
}
