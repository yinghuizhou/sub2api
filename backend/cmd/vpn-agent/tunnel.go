package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TunnelType distinguishes between VPN tunnel implementations.
type TunnelType string

const (
	TunnelTypeOpenVPN   TunnelType = "openvpn"
	TunnelTypeWireGuard TunnelType = "wireguard"
)

// TunnelInfo represents the runtime state of a tunnel.
type TunnelInfo struct {
	Name       string     `json:"name"`
	TunnelType TunnelType `json:"tunnel_type"`
	ConfigName string     `json:"config_name"`
	Region     string     `json:"region"`
	ServerIP   string     `json:"server_ip"`
	SocksPort  int        `json:"socks_port"`
	Status     string     `json:"status"`
	TunDevice  string     `json:"tun_device"`
	LocalIP    string     `json:"local_ip"`
	ExitIP     string     `json:"exit_ip"`
	Health     string     `json:"health"`
	LatencyMs  int        `json:"latency_ms"`
	Uptime     time.Time  `json:"uptime"`
	Failures   int        `json:"consecutive_failures"`
	LastCheck  time.Time  `json:"last_check"`
	ProxyID    int        `json:"proxy_id,omitempty"`
	CertID     string     `json:"cert_id,omitempty"`
}

// runningTunnel holds runtime process handles for a tunnel.
type runningTunnel struct {
	TunnelInfo
	vpnCmd        *exec.Cmd
	socksCmd      *exec.Cmd
	confPath      string
	socksConfPath string
}

// TunnelManager manages OpenVPN tunnels and 3proxy SOCKS5 proxies.
type TunnelManager struct {
	cfg        *AgentConfig
	store      *ConfigStore
	stateStore *StateStore
	callback   *CallbackClient
	certStore  *CertStore
	tunnels    map[string]*runningTunnel
	mu         sync.RWMutex
}

// NewTunnelManager creates a new TunnelManager.
func NewTunnelManager(cfg *AgentConfig, store *ConfigStore, ss *StateStore, cb *CallbackClient, cs *CertStore) *TunnelManager {
	tm := &TunnelManager{
		cfg:        cfg,
		store:      store,
		stateStore: ss,
		callback:   cb,
		certStore:  cs,
		tunnels:    make(map[string]*runningTunnel),
	}
	if err := tm.initScripts(); err != nil {
		log.Printf("Warning: failed to initialize scripts: %v", err)
	}
	return tm
}

// Create starts a new tunnel with OpenVPN + 3proxy.
func (tm *TunnelManager) Create(name, configName, region string, socksPort, proxyID int, certID string) (*TunnelInfo, error) {
	// Validate inputs before acquiring lock.
	if err := validateTunnelName(name); err != nil {
		return nil, err
	}
	if err := validateConfigName(configName); err != nil {
		return nil, err
	}

	// Phase 1: Lock, check existence, read config, setup files.
	tm.mu.Lock()
	if _, exists := tm.tunnels[name]; exists {
		tm.mu.Unlock()
		return nil, fmt.Errorf("tunnel %q already exists", name)
	}

	// Read base .ovpn content from config store.
	ovpnData, err := tm.store.Get(configName)
	if err != nil {
		tm.mu.Unlock()
		return nil, fmt.Errorf("read ovpn config %q: %w", configName, err)
	}

	// Validate and apply cert if specified.
	if certID != "" {
		if !tm.certStore.IsAvailable(certID) {
			tm.mu.Unlock()
			return nil, fmt.Errorf("cert %q in use", certID)
		}
		certPEM, keyPEM, err := tm.certStore.GetPEM(certID)
		if err != nil {
			tm.mu.Unlock()
			return nil, fmt.Errorf("get cert PEM: %w", err)
		}
		ovpnData = assembleOvpn(ovpnData, certPEM, keyPEM)
	}

	// Ensure directories exist.
	clientDir := filepath.Join(tm.cfg.StateDir, "clients")
	socksDir := filepath.Join(tm.cfg.StateDir, "3proxy")
	for _, d := range []string{clientDir, socksDir, "/run/sub2api-vpn"} {
		os.MkdirAll(d, 0755)
	}

	// Generate client config with routing directives.
	confPath := filepath.Join(clientDir, name+".conf")
	upScriptPath := filepath.Join(tm.cfg.ScriptsDir, "up.sh")
	downScriptPath := filepath.Join(tm.cfg.ScriptsDir, "down.sh")
	// Linux interface names max 15 chars; truncate tun device name to fit.
	tunDev := "tun-" + name
	if len(tunDev) > 15 {
		tunDev = tunDev[:15]
	}
	clientConf := sanitizeOvpnConfig(ovpnData) + "\n" +
		"route-nopull\n" +
		"dev " + tunDev + "\n" +
		"dev-type tun\n" +
		"script-security 2\n" +
		"setenv TUNNEL_NAME " + name + "\n" +
		"up " + upScriptPath + "\n" +
		"down " + downScriptPath + "\n"
	if err := os.WriteFile(confPath, []byte(clientConf), 0600); err != nil {
		tm.mu.Unlock()
		return nil, fmt.Errorf("write client config: %w", err)
	}

	// Start OpenVPN process.
	logFile := filepath.Join(tm.cfg.LogDir, "openvpn-"+name+".log")
	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		tm.mu.Unlock()
		return nil, fmt.Errorf("open log file: %w", err)
	}

	vpnCmd := exec.Command("openvpn", "--config", confPath)
	vpnCmd.Stdout = lf
	vpnCmd.Stderr = lf
	if err := vpnCmd.Start(); err != nil {
		lf.Close()
		tm.mu.Unlock()
		return nil, fmt.Errorf("start openvpn: %w", err)
	}
	// Close log file handle — process owns it now via fd inheritance.
	lf.Close()

	serverIP := parseServerIP(string(ovpnData))

	// Mark as "connecting" so other goroutines see it.
	rt := &runningTunnel{
		TunnelInfo: TunnelInfo{
			Name: name, TunnelType: TunnelTypeOpenVPN, ConfigName: configName, Region: region,
			ServerIP: serverIP, SocksPort: socksPort, Status: "connecting",
			TunDevice: "tun-" + name, Health: "unknown",
			Uptime: time.Now(), ProxyID: proxyID, CertID: certID,
		},
		vpnCmd:   vpnCmd,
		confPath: confPath,
	}
	tm.tunnels[name] = rt
	tm.mu.Unlock()

	// Phase 2: Wait for state file WITHOUT holding lock.
	localIP, err := waitForStateFile(name, 90*time.Second, vpnCmd)
	if err != nil {
		tm.mu.Lock()
		delete(tm.tunnels, name)
		tm.mu.Unlock()
		killProcess(vpnCmd)
		os.Remove(confPath)
		return nil, fmt.Errorf("openvpn failed to connect: %w", err)
	}

	// Phase 3: Lock again, start 3proxy, update state.
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check tunnel still exists (might have been removed while we waited).
	if _, exists := tm.tunnels[name]; !exists {
		killProcess(vpnCmd)
		os.Remove(confPath)
		return nil, fmt.Errorf("tunnel %q was removed during creation", name)
	}

	// Generate and start 3proxy.
	socksConfPath := filepath.Join(socksDir, name+".cfg")
	socksConf := fmt.Sprintf(socksConfTpl, name, localIP, socksPort)
	if err := os.WriteFile(socksConfPath, []byte(socksConf), 0644); err != nil {
		killProcess(vpnCmd)
		delete(tm.tunnels, name)
		return nil, fmt.Errorf("write 3proxy config: %w", err)
	}

	socksCmd := exec.Command("3proxy", socksConfPath)
	if err := socksCmd.Start(); err != nil {
		killProcess(vpnCmd)
		delete(tm.tunnels, name)
		return nil, fmt.Errorf("start 3proxy: %w", err)
	}

	rt.Status = "connected"
	rt.LocalIP = localIP
	rt.Health = "healthy"
	rt.socksCmd = socksCmd
	rt.socksConfPath = socksConfPath

	if certID != "" {
		tm.certStore.MarkInUse(certID, name)
	}

	tm.persistState()
	log.Printf("Tunnel %q created (socks:%d, tun:%s, ip:%s)", name, socksPort, rt.TunDevice, localIP)
	info := rt.TunnelInfo
	return &info, nil
}

// Remove stops and cleans up a tunnel completely.
func (tm *TunnelManager) Remove(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	rt, ok := tm.tunnels[name]
	if !ok {
		return fmt.Errorf("tunnel %q not found", name)
	}

	stopTunnel(rt)
	if rt.CertID != "" {
		tm.certStore.MarkFree(rt.CertID)
	}
	os.Remove(rt.confPath)
	os.Remove(rt.socksConfPath)
	delete(tm.tunnels, name)
	tm.persistState()
	log.Printf("Tunnel %q removed", name)
	return nil
}

// initScripts generates OpenVPN up/down scripts in ScriptsDir.
func (tm *TunnelManager) initScripts() error {
	if err := os.MkdirAll(tm.cfg.ScriptsDir, 0755); err != nil {
		return fmt.Errorf("create scripts dir: %w", err)
	}

	upScript := filepath.Join(tm.cfg.ScriptsDir, "up.sh")
	if err := os.WriteFile(upScript, []byte(upScriptContent), 0755); err != nil {
		return fmt.Errorf("write up.sh: %w", err)
	}

	downScript := filepath.Join(tm.cfg.ScriptsDir, "down.sh")
	if err := os.WriteFile(downScript, []byte(downScriptContent), 0755); err != nil {
		return fmt.Errorf("write down.sh: %w", err)
	}

	log.Printf("OpenVPN scripts initialized in %s", tm.cfg.ScriptsDir)
	return nil
}

const upScriptContent = `#!/bin/bash
# OpenVPN up script - called when VPN connection is established
# TUNNEL_NAME is set via "setenv TUNNEL_NAME xxx" in the OpenVPN config.
# $ifconfig_local and $4 are provided by OpenVPN with the assigned IP.

STATE_DIR="/run/sub2api-vpn"
STATE_FILE="$STATE_DIR/${TUNNEL_NAME}.state"
RT_TABLES="/etc/iproute2/rt_tables"

mkdir -p "$STATE_DIR"

# Prefer OpenVPN-provided variables over parsing ip addr
LOCAL_IP="${ifconfig_local:-$4}"
if [ -z "$LOCAL_IP" ]; then
  LOCAL_IP=$(/sbin/ip -4 addr show "$dev" 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1)
fi

if [ -z "$LOCAL_IP" ]; then
  logger -t "sub2api-vpn" "Tunnel $TUNNEL_NAME: failed to get IP" 2>/dev/null || true
  exit 1
fi

echo "LOCAL_IP=$LOCAL_IP" > "$STATE_FILE"
chmod 644 "$STATE_FILE"

# --- Policy routing: route traffic from this tunnel's IP through the VPN ---
# Truncate table name to 15 chars (same as tun device limit).
TABLE_NAME="vpn_${TUNNEL_NAME}"
TABLE_NAME="${TABLE_NAME:0:15}"

# Allocate a deterministic table ID from tunnel name (range 100-250).
TABLE_ID=$(( $(echo -n "$TUNNEL_NAME" | cksum | awk '{print $1}') % 150 + 100 ))

# Register routing table if not already present.
if ! grep -q "$TABLE_NAME" "$RT_TABLES" 2>/dev/null; then
  echo "$TABLE_ID $TABLE_NAME" >> "$RT_TABLES"
fi

# Flush stale routes and add default route through VPN device.
/sbin/ip route flush table "$TABLE_NAME" 2>/dev/null
/sbin/ip route add default dev "$dev" scope link table "$TABLE_NAME"

# Remove any existing rule for this IP, then add fresh one.
/sbin/ip rule del from "$LOCAL_IP" 2>/dev/null
/sbin/ip rule add from "$LOCAL_IP" lookup "$TABLE_NAME" priority 100

logger -t "sub2api-vpn" "Tunnel $TUNNEL_NAME connected with IP $LOCAL_IP (table $TABLE_NAME)" 2>/dev/null || true
`

const downScriptContent = `#!/bin/bash
# OpenVPN down script - called when VPN connection is closed
# TUNNEL_NAME is set via "setenv TUNNEL_NAME xxx" in the OpenVPN config.

STATE_DIR="/run/sub2api-vpn"
STATE_FILE="$STATE_DIR/${TUNNEL_NAME}.state"

# Read LOCAL_IP before removing state file.
LOCAL_IP=""
if [ -f "$STATE_FILE" ]; then
  LOCAL_IP=$(grep -oP 'LOCAL_IP=\K.*' "$STATE_FILE")
fi
rm -f "$STATE_FILE"

# Clean up policy routing rule.
if [ -n "$LOCAL_IP" ]; then
  /sbin/ip rule del from "$LOCAL_IP" 2>/dev/null
fi

TABLE_NAME="vpn_${TUNNEL_NAME}"
TABLE_NAME="${TABLE_NAME:0:15}"
/sbin/ip route flush table "$TABLE_NAME" 2>/dev/null

logger -t "sub2api-vpn" "Tunnel $TUNNEL_NAME disconnected" 2>/dev/null || true
`

// sanitizeOvpnConfig filters out macOS-incompatible OpenVPN options from
// raw .ovpn file content and replaces deprecated directives.
func sanitizeOvpnConfig(data []byte) string {
	var out strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Remove Windows-only option unsupported on macOS.
		if trimmed == "block-outside-dns" {
			continue
		}

		// Replace deprecated ns-cert-type with remote-cert-tls.
		if trimmed == "ns-cert-type server" {
			out.WriteString("remote-cert-tls server\n")
			continue
		}

		out.WriteString(line)
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}

const socksConfTpl = `log /var/log/sub2api-vpn/3proxy-%s.log D
rotate 7
timeouts 1 5 30 60 180 1800 15 60
nserver 8.8.8.8
nserver 1.1.1.1
nscache 65536
maxconn 128
external %s
auth none
socks -p%d -i0.0.0.0
`

// assembleOvpn replaces inline <cert>/<key> blocks in ovpn data with provided PEM.
func assembleOvpn(ovpnData []byte, certPEM, keyPEM string) []byte {
	content := stripInlineBlocks(string(ovpnData), "cert", "key")
	content += "\n<cert>\n" + certPEM + "\n</cert>\n"
	content += "<key>\n" + keyPEM + "\n</key>\n"
	return []byte(content)
}

// stripInlineBlocks removes <blockName>...</blockName> sections from content.
func stripInlineBlocks(content string, blockNames ...string) string {
	for _, name := range blockNames {
		re := regexp.MustCompile(`(?s)<` + name + `>.*?</` + name + `>\s*`)
		content = re.ReplaceAllString(content, "")
	}
	return content
}
