package setup

import (
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/sysutil"

	"github.com/gin-gonic/gin"
)

// installMutex prevents concurrent installation attempts (TOCTOU protection)
var installMutex sync.Mutex

// RegisterRoutes registers setup wizard routes
func RegisterRoutes(r *gin.Engine) {
	setup := r.Group("/setup")
	{
		// Status endpoint is always accessible (read-only)
		setup.GET("/status", getStatus)

		// All modification endpoints are protected by setupGuard
		protected := setup.Group("")
		protected.Use(setupGuard())
		{
			protected.POST("/test-db", testDatabase)
			protected.POST("/test-redis", testRedis)
			protected.POST("/install", install)
		}
	}
}

// SetupStatus represents the current setup state
type SetupStatus struct {
	NeedsSetup      bool     `json:"needs_setup"`
	Step            string   `json:"step"`
	ConfiguredSteps []string `json:"configured_steps"`
}

// getStatus returns the current setup status
func getStatus(c *gin.Context) {
	response.Success(c, SetupStatus{
		NeedsSetup:      NeedsSetup(),
		Step:            "welcome",
		ConfiguredSteps: GetConfiguredSteps(),
	})
}

// setupGuard middleware ensures setup endpoints are only accessible during setup mode
func setupGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !NeedsSetup() {
			response.Error(c, http.StatusForbidden, "Setup is not allowed: system is already installed")
			c.Abort()
			return
		}
		c.Next()
	}
}

// validateHostname checks if a hostname/IP is safe (no injection characters)
func validateHostname(host string) bool {
	// Allow only alphanumeric, dots, hyphens, and colons (for IPv6)
	validHost := regexp.MustCompile(`^[a-zA-Z0-9.\-:]+$`)
	return validHost.MatchString(host) && len(host) <= 253
}

// validateDBName checks if database name is safe
func validateDBName(name string) bool {
	// Allow only alphanumeric and underscores, starting with letter
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return validName.MatchString(name) && len(name) <= 63
}

// validateUsername checks if username is safe
func validateUsername(name string) bool {
	// Allow only alphanumeric and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return validName.MatchString(name) && len(name) <= 63
}

// validateEmail checks if email format is valid
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil && len(email) <= 254
}

// validatePassword checks password strength
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be at most 128 characters")
	}
	return nil
}

// validatePort checks if port is in valid range
func validatePort(port int) bool {
	return port > 0 && port <= 65535
}

// validateSSLMode checks if SSL mode is valid
func validateSSLMode(mode string) bool {
	validModes := map[string]bool{
		"disable": true, "require": true, "verify-ca": true, "verify-full": true,
	}
	return validModes[mode]
}

// TestDatabaseRequest represents database test request
type TestDatabaseRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	DBName   string `json:"dbname" binding:"required"`
	SSLMode  string `json:"sslmode"`
}

// testDatabase tests database connection
func testDatabase(c *gin.Context) {
	var req TestDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Security: Validate all inputs to prevent injection attacks
	if !validateHostname(req.Host) {
		response.Error(c, http.StatusBadRequest, "Invalid hostname format")
		return
	}
	if !validatePort(req.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid port number")
		return
	}
	if !validateUsername(req.User) {
		response.Error(c, http.StatusBadRequest, "Invalid username format")
		return
	}
	if !validateDBName(req.DBName) {
		response.Error(c, http.StatusBadRequest, "Invalid database name format")
		return
	}

	if req.SSLMode == "" {
		req.SSLMode = "disable"
	}
	if !validateSSLMode(req.SSLMode) {
		response.Error(c, http.StatusBadRequest, "Invalid SSL mode")
		return
	}

	cfg := &DatabaseConfig{
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		Password: req.Password,
		DBName:   req.DBName,
		SSLMode:  req.SSLMode,
	}

	if err := TestDatabaseConnection(cfg); err != nil {
		response.Error(c, http.StatusBadRequest, "Connection failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Connection successful"})
}

// TestRedisRequest represents Redis test request
type TestRedisRequest struct {
	Host      string `json:"host" binding:"required"`
	Port      int    `json:"port" binding:"required"`
	Password  string `json:"password"`
	DB        int    `json:"db"`
	EnableTLS bool   `json:"enable_tls"`
}

// testRedis tests Redis connection
func testRedis(c *gin.Context) {
	var req TestRedisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Security: Validate inputs
	if !validateHostname(req.Host) {
		response.Error(c, http.StatusBadRequest, "Invalid hostname format")
		return
	}
	if !validatePort(req.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid port number")
		return
	}
	if req.DB < 0 || req.DB > 15 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis database number (0-15)")
		return
	}

	cfg := &RedisConfig{
		Host:      req.Host,
		Port:      req.Port,
		Password:  req.Password,
		DB:        req.DB,
		EnableTLS: req.EnableTLS,
	}

	if err := TestRedisConnection(cfg); err != nil {
		response.Error(c, http.StatusBadRequest, "Connection failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Connection successful"})
}

// InstallRequest represents installation request.
// Database and Redis are optional when pre-configured via environment variables.
type InstallRequest struct {
	Database *DatabaseConfig `json:"database"`
	Redis    *RedisConfig    `json:"redis"`
	Admin    AdminConfig     `json:"admin" binding:"required"`
	Server   ServerConfig    `json:"server"`
}

// install performs the installation
func install(c *gin.Context) {
	// TOCTOU Protection: Acquire mutex to prevent concurrent installation
	installMutex.Lock()
	defer installMutex.Unlock()

	// Double-check after acquiring lock
	if !NeedsSetup() {
		response.Error(c, http.StatusForbidden, "Setup is not allowed: system is already installed")
		return
	}

	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Build config from env for pre-configured steps
	configuredSteps := GetConfiguredSteps()
	envCfg := BuildConfigFromEnv()

	// ========== RESOLVE DATABASE CONFIG ==========
	var dbCfg DatabaseConfig
	if req.Database != nil {
		dbCfg = *req.Database
		if !validateHostname(dbCfg.Host) {
			response.Error(c, http.StatusBadRequest, "Invalid database hostname")
			return
		}
		if !validatePort(dbCfg.Port) {
			response.Error(c, http.StatusBadRequest, "Invalid database port")
			return
		}
		if !validateUsername(dbCfg.User) {
			response.Error(c, http.StatusBadRequest, "Invalid database username")
			return
		}
		if !validateDBName(dbCfg.DBName) {
			response.Error(c, http.StatusBadRequest, "Invalid database name")
			return
		}
		if dbCfg.SSLMode == "" {
			dbCfg.SSLMode = "disable"
		}
		if !validateSSLMode(dbCfg.SSLMode) {
			response.Error(c, http.StatusBadRequest, "Invalid SSL mode")
			return
		}
	} else if containsStr(configuredSteps, "database") {
		dbCfg = envCfg.Database
	} else {
		response.Error(c, http.StatusBadRequest, "Database configuration is required")
		return
	}

	// ========== RESOLVE REDIS CONFIG ==========
	var redisCfg RedisConfig
	if req.Redis != nil {
		redisCfg = *req.Redis
		if !validateHostname(redisCfg.Host) {
			response.Error(c, http.StatusBadRequest, "Invalid Redis hostname")
			return
		}
		if !validatePort(redisCfg.Port) {
			response.Error(c, http.StatusBadRequest, "Invalid Redis port")
			return
		}
		if redisCfg.DB < 0 || redisCfg.DB > 15 {
			response.Error(c, http.StatusBadRequest, "Invalid Redis database number")
			return
		}
	} else if containsStr(configuredSteps, "redis") {
		redisCfg = envCfg.Redis
	} else {
		response.Error(c, http.StatusBadRequest, "Redis configuration is required")
		return
	}

	// ========== ADMIN VALIDATION (always required) ==========
	if !validateEmail(req.Admin.Email) {
		response.Error(c, http.StatusBadRequest, "Invalid admin email format")
		return
	}
	if err := validatePassword(req.Admin.Password); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// ========== SERVER CONFIG ==========
	serverCfg := req.Server
	if serverCfg.Host == "" {
		serverCfg.Host = envCfg.Server.Host
	}
	if serverCfg.Port == 0 {
		serverCfg.Port = envCfg.Server.Port
	}
	if serverCfg.Mode == "" {
		serverCfg.Mode = envCfg.Server.Mode
	}
	if serverCfg.Port != 0 && !validatePort(serverCfg.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid server port")
		return
	}
	if serverCfg.Mode != "release" && serverCfg.Mode != "debug" {
		response.Error(c, http.StatusBadRequest, "Invalid server mode (must be 'release' or 'debug')")
		return
	}

	// Trim whitespace from string inputs
	req.Admin.Email = strings.TrimSpace(req.Admin.Email)
	dbCfg.Host = strings.TrimSpace(dbCfg.Host)
	dbCfg.User = strings.TrimSpace(dbCfg.User)
	dbCfg.DBName = strings.TrimSpace(dbCfg.DBName)
	redisCfg.Host = strings.TrimSpace(redisCfg.Host)

	cfg := &SetupConfig{
		Database: dbCfg,
		Redis:    redisCfg,
		Admin:    req.Admin,
		Server:   serverCfg,
		JWT:      envCfg.JWT,
	}

	if err := Install(cfg); err != nil {
		response.Error(c, http.StatusInternalServerError, "Installation failed: "+err.Error())
		return
	}

	// Schedule service restart in background after sending response
	go func() {
		time.Sleep(500 * time.Millisecond)
		sysutil.RestartServiceAsync()
	}()

	response.Success(c, gin.H{
		"message": "Installation completed successfully. Service will restart automatically.",
		"restart": true,
	})
}

// containsStr checks if a string slice contains a value
func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
