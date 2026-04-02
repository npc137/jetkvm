package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jetkvm/management/middleware"
)

// tokenTTL is the lifetime of a connection token. The spec requires ≤ 60s.
const tokenTTL = 60 * time.Second

// ConnectRequest initiates a connection to a device. It is called by the SPA
// after the user clicks the "Connect" button.
//
// Flow:
//  1. Verify the calling user has an active assignment to the requested device.
//  2. Confirm the device is online.
//  3. Pre-authenticate against the JetKVM device to obtain its authToken cookie.
//  4. Store a short-lived, single-use token that bundles the JetKVM authToken.
//  5. Return the /connect/{token} URL to the client.
func ConnectRequest(db *sql.DB, publicBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := middleware.GetClaims(c)
		deviceID := c.Param("deviceId")

		// 1. Check for an active assignment for this user ↔ device pair.
		var deviceIP, devicePassword, deviceStatus string
		err := db.QueryRowContext(c.Request.Context(), `
			SELECT d.ip_address, d.password, d.status
			FROM devices d
			JOIN assignments a
			  ON a.device_id = d.id
			 AND a.user_oid  = ?
			 AND datetime('now') BETWEEN datetime(a.valid_from) AND datetime(a.valid_until)
			WHERE d.id = ?
		`, cl.OID, deviceID).Scan(&deviceIP, &devicePassword, &deviceStatus)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "no active assignment for this device"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// 2. Fail if device is offline.
		if deviceStatus != "online" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "device is offline"})
			return
		}

		// 3. Pre-authenticate against JetKVM to obtain its auth cookie.
		jetkvmAuthToken, err := preAuthenticate(deviceIP, devicePassword)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("could not authenticate to device: %v", err)})
			return
		}

		// 4. Persist the short-lived connection token.
		token := uuid.NewString()
		expiresAt := time.Now().UTC().Add(tokenTTL)
		_, err = db.ExecContext(c.Request.Context(), `
			INSERT INTO connection_tokens (token, user_oid, device_id, device_ip, jetkvm_auth_token, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, token, cl.OID, deviceID, deviceIP, jetkvmAuthToken, expiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue token"})
			return
		}

		connectURL := strings.TrimSuffix(publicBaseURL, "/") + "/connect/" + token
		c.JSON(http.StatusOK, gin.H{"connectUrl": connectURL})
	}
}

// RedeemToken handles GET /connect/:token. It validates the token, marks it
// as used, and redirects the browser to the JetKVM device via the proxy,
// injecting the pre-fetched authToken cookie.
func RedeemToken(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		now := time.Now().UTC()

		var deviceIP, jetkvmAuthToken string
		var usedAt sql.NullTime
		err := db.QueryRowContext(c.Request.Context(), `
			SELECT device_ip, jetkvm_auth_token, used_at, expires_at
			FROM connection_tokens
			WHERE token = ?
		`, token).Scan(&deviceIP, &jetkvmAuthToken, &usedAt, new(time.Time))

		if err == sql.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid token"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		var expiresAt time.Time
		err = db.QueryRowContext(c.Request.Context(),
			`SELECT expires_at FROM connection_tokens WHERE token = ?`, token,
		).Scan(&expiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		if now.After(expiresAt) {
			c.JSON(http.StatusForbidden, gin.H{"error": "token expired"})
			return
		}
		if usedAt.Valid {
			c.JSON(http.StatusForbidden, gin.H{"error": "token already used"})
			return
		}

		// Mark the token as used atomically.
		res, err := db.ExecContext(c.Request.Context(), `
			UPDATE connection_tokens SET used_at = ?
			WHERE token = ? AND used_at IS NULL
		`, now, token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			// Race: another request already consumed the token.
			c.JSON(http.StatusForbidden, gin.H{"error": "token already used"})
			return
		}

		// Inject the JetKVM authToken as a cookie and redirect to the device root
		// via the proxy path.
		c.SetCookie("authToken", jetkvmAuthToken, 3600, "/", "", false, true)
		proxyPath := "/proxy/" + url.PathEscape(deviceIP) + "/"
		c.Redirect(http.StatusFound, proxyPath)
	}
}

// ValidateToken handles GET /api/internal/token/:token — used by the Traefik
// forward-auth middleware to validate tokens before proxying traffic.
// Returns 200 with device info on success, 403 on failure.
func ValidateToken(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		now := time.Now().UTC()

		var deviceIP, jetkvmAuthToken string
		var usedAt sql.NullTime
		var expiresAt time.Time
		err := db.QueryRowContext(c.Request.Context(), `
			SELECT device_ip, jetkvm_auth_token, used_at, expires_at
			FROM connection_tokens WHERE token = ?
		`, token).Scan(&deviceIP, &jetkvmAuthToken, &usedAt, &expiresAt)

		if err == sql.ErrNoRows || now.After(expiresAt) || usedAt.Valid {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid or expired token"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"deviceIp":        deviceIP,
			"jetkvmAuthToken": jetkvmAuthToken,
		})
	}
}

// preAuthenticate calls POST /auth/login-local on the JetKVM device and
// returns the value of the authToken cookie it sets.
func preAuthenticate(deviceIP, password string) (string, error) {
	loginURL := "http://" + deviceIP + "/auth/login-local"
	body := fmt.Sprintf(`{"password":%q}`, password)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(loginURL, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("POST login: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login returned HTTP %d", resp.StatusCode)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "authToken" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("authToken cookie not found in login response")
}

// deviceStatusResponse mirrors the JSON returned by GET /device/status on a
// JetKVM device.
type deviceStatusResponse struct {
	IsSetup bool   `json:"isSetup"`
	Version string `json:"version"`
}

// PollDevice performs a single status check against a JetKVM device and
// updates the database. It is called by the background poller.
func PollDevice(db *sql.DB, deviceID, deviceIP string) {
	statusURL := "http://" + deviceIP + "/device/status"
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(statusURL)
	now := time.Now().UTC()
	if err != nil || resp.StatusCode != http.StatusOK {
		_, _ = db.Exec(`UPDATE devices SET status='offline' WHERE id=?`, deviceID)
		return
	}
	defer resp.Body.Close()

	var ds deviceStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&ds); err != nil {
		_, _ = db.Exec(`UPDATE devices SET status='offline' WHERE id=?`, deviceID)
		return
	}

	_, _ = db.Exec(`
		UPDATE devices SET status='online', last_seen_at=?, fw_version=? WHERE id=?
	`, now, ds.Version, deviceID)
}
