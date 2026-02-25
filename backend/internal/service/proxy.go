package service

import (
	"fmt"
	"time"
)

type Proxy struct {
	ID        int64
	Name      string
	Protocol  string
	Host      string
	Port      int
	Username  string
	Password  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time

	// OpenVPN / 代理分组相关字段
	Region              string
	GroupName           string
	OvpnConfig          string
	OvpnUsername        string
	OvpnPassword        string
	IsDedicated         bool
	VpnStatus           string
	VpnExitIP           string
	HealthStatus        string
	LatencyMs           *int
	LastHealthAt        *time.Time
	HealthCheckFailures int
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

// IsHealthy 检查代理是否健康可用
func (p *Proxy) IsHealthy() bool {
	return p.IsActive() && (p.HealthStatus == "" || p.HealthStatus == "healthy")
}

// IsVPN 检查是否为 OpenVPN 代理
func (p *Proxy) IsVPN() bool {
	return p.OvpnConfig != ""
}

// IsVPNConnected 检查 VPN 是否已连接
func (p *Proxy) IsVPNConnected() bool {
	return p.VpnStatus == "connected"
}

func (p *Proxy) URL() string {
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", p.Protocol, p.Username, p.Password, p.Host, p.Port)
	}
	return fmt.Sprintf("%s://%s:%d", p.Protocol, p.Host, p.Port)
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}
