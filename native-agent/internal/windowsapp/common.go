//go:build windows

package windowsapp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eitsio/eits-agent/internal/config"
)

type Status struct {
	ServiceState string
	ServerURL    string
	DeviceName   string
	Registered   bool
	LastLogLine  string
}

func ProgramDataDir() string {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "EITS", "Agent")
}

func InstallDir() string { return filepath.Join(os.Getenv("ProgramFiles"), "EITS Agent") }

func RunServiceAction(action string) error {
	allowed := map[string]bool{"start": true, "stop": true, "restart": true}
	if !allowed[action] {
		return fmt.Errorf("unsupported service action: %s", action)
	}
	if action == "restart" {
		_ = exec.Command("sc.exe", "stop", "EITSAgent").Run()
		time.Sleep(1500 * time.Millisecond)
		return exec.Command("sc.exe", "start", "EITSAgent").Run()
	}
	return exec.Command("sc.exe", action, "EITSAgent").Run()
}

func QueryStatus() Status {
	st := Status{ServiceState: "Not installed"}
	out, err := exec.Command("sc.exe", "query", "EITSAgent").CombinedOutput()
	if err == nil {
		text := string(out)
		switch {
		case strings.Contains(text, "RUNNING"):
			st.ServiceState = "Running"
		case strings.Contains(text, "STOPPED"):
			st.ServiceState = "Stopped"
		default:
			st.ServiceState = "Pending"
		}
	}
	cfgPath := config.DefaultPath()
	if cfg, err := config.Load(cfgPath); err == nil {
		st.ServerURL, st.DeviceName = cfg.ServerURL, cfg.DeviceName
		idPath := filepath.Join(cfg.StateDirectory, "identity.json")
		var id map[string]any
		if b, err := os.ReadFile(idPath); err == nil && json.Unmarshal(b, &id) == nil {
			st.Registered = id["agent_id"] != nil
		}
	}
	logPath := filepath.Join(ProgramDataDir(), "logs", "eits-agent.log")
	if b, err := os.ReadFile(logPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		if len(lines) > 0 {
			st.LastLogLine = lines[len(lines)-1]
		}
	}
	return st
}
