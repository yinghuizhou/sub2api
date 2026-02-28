package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CreateWireGuard starts a WireGuard tunnel + 3proxy SOCKS5.
func (tm *TunnelManager) CreateWireGuard(name, wgConfigName, region string, socksPort, proxyID int) (*TunnelInfo, error) {
	if err := validateTunnelName(name); err != nil {
		return nil, err
	}

	tm.mu.Lock()
	if _, exists := tm.tunnels[name]; exists {
		tm.mu.Unlock()
		return nil, fmt.Errorf("tunnel %q already exists", name)
	}

	// Read WireGuard config
	wgConfPath := filepath.Join(tm.cfg.WgConfigDir, wgConfigName)
	if _, err := os.Stat(wgConfPath); err != nil {
		tm.mu.Unlock()
		return nil, fmt.Errorf("wireguard config %q not found: %w", wgConfigName, err)
	}

	// Interface name: wg-<name>, max 15 chars
	ifName := "wg-" + name
	if len(ifName) > 15 {
		ifName = ifName[:15]
	}

	// Copy config to runtime location
	clientDir := filepath.Join(tm.cfg.StateDir, "clients")
	os.MkdirAll(clientDir, 0755)
	runtimeConf := filepath.Join(clientDir, ifName+".conf")
	wgData, err := os.ReadFile(wgConfPath)
	if err != nil {
		tm.mu.Unlock()
		return nil, fmt.Errorf("read wg config: %w", err)
	}
	if err := os.WriteFile(runtimeConf, wgData, 0600); err != nil {
		tm.mu.Unlock()
		return nil, fmt.Errorf("write runtime wg config: %w", err)
	}

	// Mark as connecting
	rt := &runningTunnel{
		TunnelInfo: TunnelInfo{
			Name: name, TunnelType: TunnelTypeWireGuard,
			ConfigName: wgConfigName, Region: region,
			SocksPort: socksPort, Status: "connecting",
			TunDevice: ifName, Health: "unknown",
			Uptime: time.Now(), ProxyID: proxyID,
		},
		confPath: runtimeConf,
	}
	tm.tunnels[name] = rt
	tm.mu.Unlock()

	// Bring up WireGuard interface
	cmd := exec.Command("wg-quick", "up", runtimeConf)
	if output, err := cmd.CombinedOutput(); err != nil {
		tm.mu.Lock()
		delete(tm.tunnels, name)
		tm.mu.Unlock()
		os.Remove(runtimeConf)
		return nil, fmt.Errorf("wg-quick up failed: %w\nOutput: %s", err, output)
	}

	// Get local IP from WireGuard interface
	localIP, err := getWgLocalIP(ifName)
	if err != nil {
		exec.Command("wg-quick", "down", runtimeConf).Run()
		tm.mu.Lock()
		delete(tm.tunnels, name)
		tm.mu.Unlock()
		os.Remove(runtimeConf)
		return nil, fmt.Errorf("get wg local IP: %w", err)
	}

	// Start 3proxy
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tunnels[name]; !exists {
		exec.Command("wg-quick", "down", runtimeConf).Run()
		return nil, fmt.Errorf("tunnel %q removed during creation", name)
	}

	socksDir := filepath.Join(tm.cfg.StateDir, "3proxy")
	os.MkdirAll(socksDir, 0755)
	socksConfPath := filepath.Join(socksDir, name+".cfg")
	socksConf := fmt.Sprintf(socksConfTpl, name, localIP, socksPort)
	if err := os.WriteFile(socksConfPath, []byte(socksConf), 0644); err != nil {
		exec.Command("wg-quick", "down", runtimeConf).Run()
		delete(tm.tunnels, name)
		return nil, fmt.Errorf("write 3proxy config: %w", err)
	}

	socksCmd := exec.Command("3proxy", socksConfPath)
	if err := socksCmd.Start(); err != nil {
		exec.Command("wg-quick", "down", runtimeConf).Run()
		delete(tm.tunnels, name)
		return nil, fmt.Errorf("start 3proxy: %w", err)
	}

	rt.Status = "connected"
	rt.LocalIP = localIP
	rt.Health = "healthy"
	rt.socksCmd = socksCmd
	rt.socksConfPath = socksConfPath

	tm.persistState()
	log.Printf("WireGuard tunnel %q created (socks:%d, if:%s, ip:%s)",
		name, socksPort, ifName, localIP)
	info := rt.TunnelInfo
	return &info, nil
}

// getWgLocalIP reads the local IP assigned to a WireGuard interface.
func getWgLocalIP(ifName string) (string, error) {
	time.Sleep(2 * time.Second)

	out, err := exec.Command("ip", "-4", "addr", "show", ifName).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ip addr show %s: %w\nOutput: %s", ifName, err, out)
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.Split(parts[1], "/")[0]
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("no IPv4 address found on %s", ifName)
}
