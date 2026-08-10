package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/eitsio/eits-agent/internal/collector"
	"github.com/eitsio/eits-agent/internal/config"
	"github.com/eitsio/eits-agent/internal/queue"
	"github.com/eitsio/eits-agent/internal/transport"
)

type Runtime struct {
	ConfigPath    string
	Config        config.Config
	Version       string
	Client        *transport.Client
	Collector     *collector.Collector
	Queue         queue.Queue
	Logger        *log.Logger
	lastInventory time.Time
}

type IdentityFile struct {
	AgentID     string    `json:"agent_id"`
	AgentSecret string    `json:"agent_secret"`
	EnrolledAt  time.Time `json:"enrolled_at"`
}

type metricsPayload struct {
	collector.Snapshot
	PortResults []transport.PortResult `json:"port_results"`
}

func New(configPath, version string, logger *log.Logger) (*Runtime, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	client := transport.New(cfg.ServerURL, time.Duration(cfg.RequestTimeoutSeconds)*time.Second, cfg.SkipTLSVerify)
	r := &Runtime{ConfigPath: configPath, Config: cfg, Version: version, Client: client, Collector: collector.New(), Logger: logger}
	r.Queue = queue.Queue{Path: filepath.Join(cfg.StateDirectory, "metrics.queue"), MaximumRecords: cfg.Queue.MaximumRecords}
	return r, nil
}
func (r *Runtime) identityPath() string {
	return filepath.Join(r.Config.StateDirectory, "identity.json")
}
func (r *Runtime) loadIdentity() (transport.Identity, error) {
	var file IdentityFile
	data, err := os.ReadFile(r.identityPath())
	if err != nil {
		return transport.Identity{}, err
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return transport.Identity{}, err
	}
	if file.AgentID == "" || file.AgentSecret == "" {
		return transport.Identity{}, errors.New("identity file is incomplete")
	}
	return transport.Identity{AgentID: file.AgentID, AgentSecret: file.AgentSecret}, nil
}
func (r *Runtime) saveIdentity(identity transport.Identity) error {
	if err := os.MkdirAll(r.Config.StateDirectory, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(IdentityFile{AgentID: identity.AgentID, AgentSecret: identity.AgentSecret, EnrolledAt: time.Now().UTC()}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.identityPath(), data, 0600)
}
func (r *Runtime) ensureIdentity() error {
	identity, err := r.loadIdentity()
	if err == nil {
		r.Client.Identity = identity
		return nil
	}
	if r.Config.EnrollmentToken == "" {
		return fmt.Errorf("no valid identity and no enrollment_token in %s", r.ConfigPath)
	}
	host, _ := os.Hostname()
	name := r.Config.DeviceName
	if name == "" {
		name = host
	}
	payload := map[string]any{"enrollment_token": r.Config.EnrollmentToken, "name": name, "hostname": host, "os": runtime.GOOS, "architecture": runtime.GOARCH, "agent_version": r.Version}
	r.Logger.Printf("No identity found; enrolling %s", name)
	identity, err = r.Client.Enroll(payload)
	if err != nil {
		return err
	}
	if err := r.saveIdentity(identity); err != nil {
		return err
	}
	r.Client.Identity = identity
	// Remove enrollment token after successful enrollment.
	r.Config.EnrollmentToken = ""
	if err := config.Save(r.ConfigPath, r.Config); err != nil {
		r.Logger.Printf("warning: could not remove enrollment token: %v", err)
	}
	return nil
}
func (r *Runtime) sendInventory(force bool) error {
	if !force && time.Since(r.lastInventory) < 24*time.Hour {
		return nil
	}
	inv, err := r.Collector.Inventory()
	if err != nil {
		return err
	}
	data, err := json.Marshal(inv)
	if err != nil {
		return err
	}
	if err := r.Client.PostInventory(data); err != nil {
		return err
	}
	r.lastInventory = time.Now()
	r.Logger.Printf("inventory reported: %s %s, %s, %d cores/%d threads, %d GPU(s)", inv.Manufacturer, inv.Model, inv.CPUModel, inv.CPUPhysicalCores, inv.CPULogicalProcessors, len(inv.GPUs))
	return nil
}

func (r *Runtime) collectPayload() ([]byte, int, error) {
	cfg, err := r.Client.GetConfig()
	if err != nil {
		return nil, 0, err
	}
	snapshot, err := r.Collector.Collect(r.Version)
	if err != nil {
		return nil, 0, err
	}
	results := make([]transport.PortResult, 0, len(cfg.PortChecks))
	for _, check := range cfg.PortChecks {
		results = append(results, transport.CheckPort(check))
	}
	payload := metricsPayload{Snapshot: snapshot, PortResults: results}
	data, err := json.Marshal(payload)
	return data, len(results), err
}
func (r *Runtime) flushQueue() {
	if !r.Config.Queue.Enabled {
		return
	}
	rows, err := r.Queue.ReadAll()
	if err != nil {
		r.Logger.Printf("queue read error: %v", err)
		return
	}
	sent := 0
	for _, row := range rows {
		if err := r.Client.PostMetrics(row); err != nil {
			break
		}
		sent++
	}
	if sent > 0 {
		_ = r.Queue.Replace(rows[sent:])
		r.Logger.Printf("uploaded %d queued metric sample(s)", sent)
	}
}
func (r *Runtime) Once() error {
	if err := r.ensureIdentity(); err != nil {
		return err
	}
	r.flushQueue()
	if err := r.sendInventory(false); err != nil {
		r.Logger.Printf("inventory error: %v", err)
	}
	data, ports, err := r.collectPayload()
	if err != nil {
		return err
	}
	if err := r.Client.PostMetrics(data); err != nil {
		if r.Config.Queue.Enabled {
			if qerr := r.Queue.Add(data); qerr != nil {
				r.Logger.Printf("queue write error: %v", qerr)
			}
		}
		return err
	}
	var payload metricsPayload
	_ = json.Unmarshal(data, &payload)
	r.Logger.Printf("reported cpu=%.1f%% memory=%.1f%% disks=%d ports=%d processes=%d", payload.CPUPercent, payload.MemoryPercent, len(payload.Disks), ports, len(payload.Processes))
	return nil
}
func (r *Runtime) Run(stop <-chan struct{}) error {
	if err := r.ensureIdentity(); err != nil {
		return err
	}
	r.Logger.Printf("EITS agent %s started as %s", r.Version, r.Client.Identity.AgentID)
	if err := r.sendInventory(true); err != nil {
		r.Logger.Printf("inventory error: %v", err)
	}
	// Prime CPU counters before first report.
	_, _ = r.Collector.Collect(r.Version)
	time.Sleep(time.Second)
	ticker := time.NewTicker(time.Duration(r.Config.CollectionIntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		if err := r.Once(); err != nil {
			r.Logger.Printf("report error: %v", err)
		}
		select {
		case <-stop:
			r.Logger.Printf("EITS agent stopping")
			return nil
		case <-ticker.C:
		}
	}
}
func (r *Runtime) Diagnostics() error {
	if err := r.ensureIdentity(); err != nil {
		return err
	}
	cfg, err := r.Client.GetConfig()
	if err != nil {
		return fmt.Errorf("API configuration test failed: %w", err)
	}
	snapshot, err := r.Collector.Collect(r.Version)
	if err != nil {
		return fmt.Errorf("collector test failed: %w", err)
	}
	r.Logger.Printf("Server: %s", r.Config.ServerURL)
	r.Logger.Printf("Identity: %s", r.Client.Identity.AgentID)
	r.Logger.Printf("Host: %s %s/%s", snapshot.Hostname, snapshot.OS, snapshot.Architecture)
	r.Logger.Printf("CPU %.1f%%, memory %.1f%%, disks %d, configured checks %d", snapshot.CPUPercent, snapshot.MemoryPercent, len(snapshot.Disks), len(cfg.PortChecks))
	return nil
}
