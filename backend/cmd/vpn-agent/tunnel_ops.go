package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// List returns info for all tunnels.
func (tm *TunnelManager) List() []TunnelInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	out := make([]TunnelInfo, 0, len(tm.tunnels))
	for _, rt := range tm.tunnels {
		out = append(out, rt.TunnelInfo)
	}
	return out
}

// Get returns info for a single tunnel by name.
func (tm *TunnelManager) Get(name string) *TunnelInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if rt, ok := tm.tunnels[name]; ok {
		info := rt.TunnelInfo
		return &info
	}
	return nil
}

// Restart stops and re-creates a tunnel with the same config.
func (tm *TunnelManager) Restart(name string) error {
	tm.mu.Lock()
	rt, ok := tm.tunnels[name]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("tunnel %q not found", name)
	}
	cfg := rt.ConfigName
	region := rt.Region
	port := rt.SocksPort
	pid := rt.ProxyID
	certID := rt.CertID

	// Stop and remove under lock
	stopTunnel(rt)
	if certID != "" {
		tm.certStore.MarkFree(certID)
	}
	os.Remove(rt.confPath)
	os.Remove(rt.socksConfPath)
	delete(tm.tunnels, name)
	tm.persistState()
	tm.mu.Unlock()

	// Re-create (Create handles its own locking)
	_, err := tm.Create(name, cfg, region, port, pid, certID)
	return err
}

// Stop terminates processes but keeps tunnel in map as "disconnected".
func (tm *TunnelManager) Stop(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	rt, ok := tm.tunnels[name]
	if !ok {
		return fmt.Errorf("tunnel %q not found", name)
	}

	stopTunnel(rt)
	rt.Status = "disconnected"
	rt.Health = "unhealthy"
	log.Printf("Tunnel %q stopped", name)
	return nil
}

// Start restarts a stopped tunnel from its saved config.
func (tm *TunnelManager) Start(name string) error {
	tm.mu.Lock()
	rt, ok := tm.tunnels[name]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("tunnel %q not found", name)
	}
	if rt.Status == "connected" {
		tm.mu.Unlock()
		return nil
	}
	cfg := rt.ConfigName
	region := rt.Region
	port := rt.SocksPort
	pid := rt.ProxyID
	certID := rt.CertID
	if certID != "" {
		tm.certStore.MarkFree(certID)
	}
	delete(tm.tunnels, name)
	tm.mu.Unlock()

	_, err := tm.Create(name, cfg, region, port, pid, certID)
	return err
}

// StopAll gracefully stops all tunnels.
func (tm *TunnelManager) StopAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for name, rt := range tm.tunnels {
		stopTunnel(rt)
		log.Printf("Tunnel %q stopped (shutdown)", name)
	}
}

// RestoreFromState re-creates tunnels saved in state.json.
func (tm *TunnelManager) RestoreFromState() error {
	state, err := tm.stateStore.Load()
	if err != nil {
		return err
	}
	for _, ts := range state.Tunnels {
		log.Printf("Restoring tunnel %q (%s port %d)", ts.Name, ts.ConfigName, ts.SocksPort)
		if _, err := tm.Create(ts.Name, ts.ConfigName, ts.Region, ts.SocksPort, ts.ProxyID, ts.CertID); err != nil {
			log.Printf("Warning: failed to restore tunnel %q: %v", ts.Name, err)
		}
	}
	return nil
}

// SwitchConfig stops a tunnel and restarts it with a new .ovpn config.
func (tm *TunnelManager) SwitchConfig(name, newConfigName string) error {
	tm.mu.Lock()
	rt, ok := tm.tunnels[name]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("tunnel %q not found", name)
	}
	region := rt.Region
	port := rt.SocksPort
	pid := rt.ProxyID
	certID := rt.CertID

	stopTunnel(rt)
	if certID != "" {
		tm.certStore.MarkFree(certID)
	}
	os.Remove(rt.confPath)
	os.Remove(rt.socksConfPath)
	delete(tm.tunnels, name)
	tm.persistState()
	tm.mu.Unlock()

	_, err := tm.Create(name, newConfigName, region, port, pid, certID)
	return err
}

// NextAvailablePort finds the next unused SOCKS port starting from 10801.
func (tm *TunnelManager) NextAvailablePort() (int, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	used := make(map[int]bool)
	for _, rt := range tm.tunnels {
		used[rt.SocksPort] = true
	}
	for port := 10801; port < 10900; port++ {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range 10801-10899")
}

// persistState saves current tunnels to state.json. Caller must hold lock.
func (tm *TunnelManager) persistState() {
	state := &AgentState{Tunnels: make(map[string]TunnelState)}
	for name, rt := range tm.tunnels {
		state.Tunnels[name] = TunnelState{
			Name: name, ConfigName: rt.ConfigName,
			SocksPort: rt.SocksPort, Region: rt.Region,
			ServerIP: rt.ServerIP, ProxyID: rt.ProxyID,
			CertID: rt.CertID,
		}
	}
	if err := tm.stateStore.Save(state); err != nil {
		log.Printf("Error persisting state: %v", err)
	}
}

// waitForStateFile polls for the up.sh state file and reads LOCAL_IP.
// It also checks if the VPN process has exited early.
func waitForStateFile(name string, timeout time.Duration, vpnCmd *exec.Cmd) (string, error) {
	stateFile := "/run/sub2api-vpn/" + name + ".state"
	deadline := time.Now().Add(timeout)

	// Channel to detect early process exit
	exited := make(chan error, 1)
	go func() { exited <- vpnCmd.Wait() }()

	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			return "", fmt.Errorf("openvpn process exited early: %v", err)
		default:
		}
		ip, err := readStateFile(stateFile)
		if err == nil && ip != "" {
			return ip, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("timeout waiting for %s", stateFile)
}

// readStateFile parses LOCAL_IP from a vpn state file.
func readStateFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "LOCAL_IP=") {
			return strings.TrimPrefix(line, "LOCAL_IP="), nil
		}
	}
	return "", fmt.Errorf("LOCAL_IP not found in %s", path)
}

// stopTunnel sends SIGTERM to both processes, with SIGKILL fallback.
func stopTunnel(rt *runningTunnel) {
	for _, label := range []struct {
		name string
		cmd  *exec.Cmd
	}{{"openvpn", rt.vpnCmd}, {"3proxy", rt.socksCmd}} {
		if label.cmd == nil || label.cmd.Process == nil {
			continue
		}
		_ = label.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func(c *exec.Cmd) { done <- c.Wait() }(label.cmd)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = label.cmd.Process.Signal(syscall.SIGKILL)
			<-done
		}
	}
}

// killProcess forcefully kills a single process.
func killProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}
