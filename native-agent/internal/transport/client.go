package transport

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Identity struct {
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
}

type PortCheck struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Protocol       string `json:"protocol"`
	UDPPayload     string `json:"udp_payload"`
	ExpectResponse bool   `json:"expect_response"`
}

type AgentConfig struct {
	Revision   int         `json:"revision"`
	PortChecks []PortCheck `json:"port_checks"`
}

type PortResult struct {
	CheckID   int     `json:"check_id"`
	IsUp      bool    `json:"is_up"`
	LatencyMS float64 `json:"latency_ms"`
	Error     string  `json:"error"`
}

type Client struct {
	BaseURL  string
	HTTP     *http.Client
	Identity Identity
}

func New(baseURL string, timeout time.Duration, skipTLSVerify bool) *Client {
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: skipTLSVerify}}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: timeout, Transport: transport}}
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("X-Agent-ID", c.Identity.AgentID)
	req.Header.Set("X-Agent-Secret", c.Identity.AgentSecret)
}

func decodeError(resp *http.Response) error {
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if len(detail) > 0 {
		return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}
	return fmt.Errorf("server returned %s", resp.Status)
}

func (c *Client) Enroll(payload any) (Identity, error) {
	var identity Identity
	body, err := json.Marshal(payload)
	if err != nil {
		return identity, err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/agent/enroll", bytes.NewReader(body))
	if err != nil {
		return identity, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return identity, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return identity, decodeError(resp)
	}
	err = json.NewDecoder(resp.Body).Decode(&identity)
	return identity, err
}

func (c *Client) GetConfig() (AgentConfig, error) {
	var cfg AgentConfig
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/agent/config", nil)
	if err != nil {
		return cfg, err
	}
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return cfg, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return cfg, decodeError(resp)
	}
	err = json.NewDecoder(resp.Body).Decode(&cfg)
	if cfg.PortChecks == nil {
		cfg.PortChecks = []PortCheck{}
	}
	return cfg, err
}

func (c *Client) PostMetrics(payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/agent/metrics", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeError(resp)
	}
	return nil
}

func (c *Client) PostInventory(payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/agent/inventory", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return decodeError(resp)
	}
	return nil
}

func CheckPort(check PortCheck) PortResult {
	start := time.Now()
	result := PortResult{CheckID: check.ID}
	timeout := time.Duration(check.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	address := net.JoinHostPort(check.Host, strconv.Itoa(check.Port))
	protocol := strings.ToLower(strings.TrimSpace(check.Protocol))
	if protocol == "udp" {
		conn, err := net.DialTimeout("udp", address, timeout)
		if err != nil {
			result.Error = err.Error()
			result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
			return result
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(timeout))
		payload := []byte(check.UDPPayload)
		if len(payload) == 0 {
			payload = []byte{0}
		}
		if _, err = conn.Write(payload); err != nil {
			result.Error = err.Error()
			result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
			return result
		}
		buffer := make([]byte, 2048)
		_, err = conn.Read(buffer)
		result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
		if err == nil {
			result.IsUp = true
			return result
		}
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() && !check.ExpectResponse {
			result.IsUp = true
			return result
		}
		result.Error = err.Error()
		return result
	}
	conn, err := net.DialTimeout("tcp", address, timeout)
	result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.IsUp = true
	_ = conn.Close()
	return result
}
