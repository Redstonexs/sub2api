package setup

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/sysutil"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// installMutex prevents concurrent installation attempts (TOCTOU protection)
var installMutex sync.Mutex

// bootstrapTokenState holds the installation-scoped credential for the HTTP setup wizard.
// It is set by runSetupServer before registering routes and never by CLI/AUTO_SETUP.
var bootstrapTokenState struct {
	sync.RWMutex
	value string
}

const (
	setupMutationMaxBodyBytes = 64 << 10
	setupMutationRequests     = 20
)

// SetBootstrapToken sets the global bootstrap token for the HTTP setup wizard.
// Called only from runSetupServer; never from CLI or AUTO_SETUP.
func SetBootstrapToken(token string) {
	bootstrapTokenState.Lock()
	bootstrapTokenState.value = token
	bootstrapTokenState.Unlock()
}

func currentBootstrapToken() string {
	bootstrapTokenState.RLock()
	defer bootstrapTokenState.RUnlock()
	return bootstrapTokenState.value
}

// RegisterRoutes registers setup wizard routes
func RegisterRoutes(r *gin.Engine) {
	registerRoutes(r, rate.NewLimiter(rate.Every(time.Minute/setupMutationRequests), setupMutationRequests), setupMutationMaxBodyBytes)
}

func registerRoutes(r *gin.Engine, limiter *rate.Limiter, maxBodyBytes int64) {
	setup := r.Group("/setup")
	// Peer/Host checks apply to ALL /setup routes (including GET /status).
	// POST-only checks (Origin, Sec-Fetch-Site, Content-Type) are inside
	// setupRequestGate and only activate on POST.
	setup.Use(setupRequestGate())
	{
		// Status endpoint: read-only, peer/Host checked but no token required.
		setup.GET("/status", getStatus)

		// All modification endpoints are protected by:
		//   1. setupGuard — system is in setup mode
		//   2. setupBootstrapToken — token verification (before rate limiter)
		//   3. setupRequestBodyLimit — body size limit
		//   4. setupMutationRateLimit — rate limiter
		protected := setup.Group("")
		protected.Use(setupGuard(), setupBootstrapToken(), setupRequestBodyLimit(maxBodyBytes), setupMutationRateLimit(limiter))
		{
			protected.POST("/test-db", testDatabase)
			protected.POST("/test-redis", testRedis)
			protected.POST("/install", install)
		}
	}
}

// setupRequestGate applies peer/Host checks to all setup-server routes, and
// Origin/fetch-site/Content-Type checks to every setup POST. It does not rely
// on the generic CORS middleware or trust forwarded headers.
func setupRequestGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ====== Peer (RemoteAddr) check ======
		// Require the TCP peer to be an IP loopback after IPv4 unmapping.
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			abortRequestGate(c)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil {
			abortRequestGate(c)
			return
		}
		// Unmap IPv4-in-IPv6 to get the underlying IPv4.
		ip = ip.To16()
		if ip == nil {
			abortRequestGate(c)
			return
		}
		// Check if it is any loopback address.
		if !ip.IsLoopback() {
			abortRequestGate(c)
			return
		}

		// ====== Host check ======
		// Host must be exactly "localhost" (case-insensitive) or a literal
		// loopback IP, with an optional valid port. Reject suffixes, userinfo,
		// zones, or malformed authority.
		if !isValidLoopbackHost(c.Request.Host) {
			abortRequestGate(c)
			return
		}

		// ====== POST-only checks ======
		if c.Request.Method == http.MethodPost {
			// Origin check: must be present and exactly one, with http/https
			// scheme, no credentials/path/query/fragment, and loopback authority
			// that canonicalizes to the same as Host.
			origins := c.Request.Header.Values("Origin")
			if len(origins) != 1 {
				abortRequestGate(c)
				return
			}
			if !isValidLoopbackOrigin(origins[0], c.Request.Host) {
				abortRequestGate(c)
				return
			}

			// Sec-Fetch-Site check: if present, must be "same-origin".
			if fetchSite := c.Request.Header.Get("Sec-Fetch-Site"); fetchSite != "" {
				if strings.ToLower(strings.TrimSpace(fetchSite)) != "same-origin" {
					abortRequestGate(c)
					return
				}
			}

			// Content-Type check: must parse as application/json (parameters permitted).
			ct := c.Request.Header.Get("Content-Type")
			if ct == "" || !isJSONContentType(ct) {
				c.Abort()
				response.Error(c, http.StatusUnsupportedMediaType, "Unsupported Media Type")
				return
			}
		}

		c.Next()
	}
}

// abortRequestGate aborts with a generic 403 to avoid leaking information.
func abortRequestGate(c *gin.Context) {
	c.Abort()
	response.Error(c, http.StatusForbidden, "Forbidden")
}

// isJSONContentType checks whether the Content-Type header parses as application/json.
func isJSONContentType(rawCT string) bool {
	mediaType, _, err := mime.ParseMediaType(rawCT)
	return err == nil && mediaType == "application/json"
}

// parseLoopbackAuthority parses a strictly formed loopback Host/Origin authority.
// It returns canonical host and optional port values for safe equality comparison.
func parseLoopbackAuthority(authority string) (host, port string, ok bool) {
	if authority == "" || strings.ContainsAny(authority, "@%/?#") {
		return "", "", false
	}

	switch {
	case strings.HasPrefix(authority, "[") || strings.Contains(authority, "]"):
		if !strings.HasPrefix(authority, "[") || strings.Count(authority, "[") != 1 || strings.Count(authority, "]") != 1 {
			return "", "", false
		}
		end := strings.IndexByte(authority, ']')
		if end <= 1 {
			return "", "", false
		}
		host = authority[1:end]
		suffix := authority[end+1:]
		if suffix != "" {
			if !strings.HasPrefix(suffix, ":") || len(suffix) == 1 {
				return "", "", false
			}
			port = suffix[1:]
		}
		if net.ParseIP(host) == nil {
			return "", "", false
		}
	case strings.Count(authority, ":") == 1:
		var err error
		host, port, err = net.SplitHostPort(authority)
		if err != nil || host == "" || port == "" {
			return "", "", false
		}
	default:
		host = authority
	}

	if port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return "", "", false
		}
	}

	if strings.EqualFold(host, "localhost") {
		return "localhost", port, true
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", "", false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String(), port, true
	}
	return ip.String(), port, true
}

// isValidLoopbackHost checks whether hostPort is a strictly formed loopback authority.
func isValidLoopbackHost(hostPort string) bool {
	_, _, ok := parseLoopbackAuthority(hostPort)
	return ok
}

// isValidLoopbackOrigin checks whether the origin is a valid http/https URL
// with a loopback authority that canonicalizes to the same as hostPort.
// Accepts "[::1]" syntax even though the listener is IPv4-only.
func isValidLoopbackOrigin(origin, hostPort string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Must be http or https.
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// No credentials, path, query, or fragment allowed.
	if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}

	// The origin authority must be a loopback host.
	if !isValidLoopbackHost(u.Host) {
		return false
	}

	// Normalize: strip the scheme-default port so that:
	//   http://localhost:80  → localhost (default port for http)
	//   https://localhost:443 → localhost (default port for https)
	//   http://localhost:443 → localhost:443 (non-default — still kept)
	// Then compare to the Host header, applying the same default-port stripping.
	// Both sides are bracket-normalized so [::1] vs ::1 comparisons work.
	defaultPort := "80"
	if u.Scheme == "https" {
		defaultPort = "443"
	}
	originNorm := stripDefaultPort(u.Host, defaultPort)
	requestNorm := stripDefaultPort(hostPort, defaultPort)
	return originNorm == requestNorm
}

// stripDefaultPort removes the given defaultPort from an authority for comparison.
// It also normalizes brackets on IPv6 addresses so that "[::1]:80" compares
// equal to "::1" (after default port stripping).
func stripDefaultPort(authority, defaultPort string) string {
	host, port, ok := parseLoopbackAuthority(authority)
	if !ok {
		return ""
	}
	if port == "" || port == defaultPort {
		return host
	}
	return host + ":" + port
}

// setupBootstrapToken verifies the X-Bootstrap-Token header on POST mutations.
// It returns a generic 403 on missing or invalid token.
// This runs before the rate limiter so attackers cannot consume the operator quota.
func setupBootstrapToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedToken := currentBootstrapToken()
		// If no token is set (CLI/AUTO_SETUP path), skip this middleware.
		// This should not happen because the HTTP setup path always sets the token,
		// but guard defensively.
		if expectedToken == "" {
			response.Error(c, http.StatusForbidden, "Forbidden")
			c.Abort()
			return
		}

		// Exactly one token header is required.
		tokens := c.Request.Header.Values(BootstrapTokenHeader)
		if len(tokens) != 1 {
			response.Error(c, http.StatusForbidden, "Forbidden")
			c.Abort()
			return
		}

		// Constant-time comparison to prevent timing side channels.
		if !constantTimeCompare(tokens[0], expectedToken) {
			response.Error(c, http.StatusForbidden, "Forbidden")
			c.Abort()
			return
		}

		c.Next()
	}
}

func setupRequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && c.Request != nil && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

func setupMutationRateLimit(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || limiter.Allow() {
			c.Next()
			return
		}
		response.Error(c, http.StatusTooManyRequests, "Too many setup requests")
		c.Abort()
	}
}

// SetupStatus represents the current setup state
type SetupStatus struct {
	NeedsSetup bool   `json:"needs_setup"`
	Step       string `json:"step"`
}

// getStatus returns the current setup status
func getStatus(c *gin.Context) {
	response.Success(c, SetupStatus{
		NeedsSetup: NeedsSetup(),
		Step:       "welcome",
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
	Username  string `json:"username"`
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
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) > 128 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis username")
		return
	}

	cfg := &RedisConfig{
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
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

// InstallRequest represents installation request
type InstallRequest struct {
	Database DatabaseConfig `json:"database" binding:"required"`
	Redis    RedisConfig    `json:"redis" binding:"required"`
	Admin    AdminConfig    `json:"admin" binding:"required"`
	Server   ServerConfig   `json:"server"`
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

	req.Admin.Email = strings.TrimSpace(req.Admin.Email)
	req.Database.Host = strings.TrimSpace(req.Database.Host)
	req.Database.User = strings.TrimSpace(req.Database.User)
	req.Database.DBName = strings.TrimSpace(req.Database.DBName)
	req.Redis.Host = strings.TrimSpace(req.Redis.Host)
	req.Redis.Username = strings.TrimSpace(req.Redis.Username)

	// ========== COMPREHENSIVE INPUT VALIDATION ==========
	// Database validation
	if !validateHostname(req.Database.Host) {
		response.Error(c, http.StatusBadRequest, "Invalid database hostname")
		return
	}
	if !validatePort(req.Database.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid database port")
		return
	}
	if !validateUsername(req.Database.User) {
		response.Error(c, http.StatusBadRequest, "Invalid database username")
		return
	}
	if !validateDBName(req.Database.DBName) {
		response.Error(c, http.StatusBadRequest, "Invalid database name")
		return
	}

	// Redis validation
	if !validateHostname(req.Redis.Host) {
		response.Error(c, http.StatusBadRequest, "Invalid Redis hostname")
		return
	}
	if !validatePort(req.Redis.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid Redis port")
		return
	}
	if req.Redis.DB < 0 || req.Redis.DB > 15 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis database number")
		return
	}
	if len(req.Redis.Username) > 128 {
		response.Error(c, http.StatusBadRequest, "Invalid Redis username")
		return
	}

	// Admin validation
	if !validateEmail(req.Admin.Email) {
		response.Error(c, http.StatusBadRequest, "Invalid admin email format")
		return
	}
	if err := validatePassword(req.Admin.Password); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Server validation
	if req.Server.Port != 0 && !validatePort(req.Server.Port) {
		response.Error(c, http.StatusBadRequest, "Invalid server port")
		return
	}

	// ========== SET DEFAULTS ==========
	if req.Database.SSLMode == "" {
		req.Database.SSLMode = "disable"
	}
	if !validateSSLMode(req.Database.SSLMode) {
		response.Error(c, http.StatusBadRequest, "Invalid SSL mode")
		return
	}
	if req.Server.Host == "" {
		req.Server.Host = "127.0.0.1"
	}
	if req.Server.Port == 0 {
		req.Server.Port = 8080
	}
	if req.Server.Mode == "" {
		req.Server.Mode = "release"
	}
	// Validate server mode
	if req.Server.Mode != "release" && req.Server.Mode != "debug" {
		response.Error(c, http.StatusBadRequest, "Invalid server mode (must be 'release' or 'debug')")
		return
	}

	cfg := &SetupConfig{
		Database: req.Database,
		Redis:    req.Redis,
		Admin:    req.Admin,
		Server:   req.Server,
		JWT: JWTConfig{
			ExpireHour: 24,
		},
	}

	if err := Install(cfg); err != nil {
		response.Error(c, http.StatusInternalServerError, "Installation failed: "+err.Error())
		return
	}

	// Schedule service restart in background after sending response
	// This ensures the client receives the success response before the service restarts
	go func() {
		// Wait a moment to ensure the response is sent
		time.Sleep(500 * time.Millisecond)
		sysutil.RestartServiceAsync()
	}()

	response.Success(c, gin.H{
		"message": "Installation completed successfully. Service will restart automatically.",
		"restart": true,
	})
}
