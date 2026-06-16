package frp

import (
    "os/exec"
    "testing"
    "time"
    "bytes"
    "errors"
    "github.com/asyou/server/internal/model"
)

// fakeCmd simulates a running command
type fakeCmd struct {
    done chan error
    stdout *bytes.Buffer
    stderr *bytes.Buffer
}

func (f *fakeCmd) Start() error { return nil }
func (f *fakeCmd) Wait() error { return <-f.done }
func (f *fakeCmd) Kill() error { return errors.New("kill-not-supported") }

// We can't easily implement exec.Cmd interface; instead override CmdBuilder to return a real exec.Cmd calling `sleep`.

func TestManagerStartStop(t *testing.T) {
    m := NewManager()
    // override CmdBuilder to run `sleep 1` so process exits quickly
    m.CmdBuilder = func(cfgPath string) *exec.Cmd {
        return exec.Command("/bin/sh", "-c", "sleep 1")
    }

    proxy := &model.Proxy{ID: 99, LocalIP: "127.0.0.1", LocalPort: 8080, Type: "tcp"}
    node := &model.Node{ID: 1, Host: "127.0.0.1", BindPort: 7000}

    if err := m.Start(proxy, node); err != nil {
        t.Fatalf("start failed: %v", err)
    }
    if !m.IsRunning(proxy.ID) {
        t.Fatalf("expected running")
    }
    // wait for process to exit
    time.Sleep(1500 * time.Millisecond)
    if m.IsRunning(proxy.ID) {
        t.Fatalf("expected not running after sleep")
    }

    // start again and stop
    if err := m.Start(proxy, node); err != nil {
        t.Fatalf("start failed: %v", err)
    }
    if !m.IsRunning(proxy.ID) {
        t.Fatalf("expected running")
    }
    if err := m.Stop(proxy.ID); err != nil {
        // Stop may kill; allow error
        t.Logf("stop returned error: %v", err)
    }
}
