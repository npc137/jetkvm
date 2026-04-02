// Package middleware provides Gin middleware for the management backend.
package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/jetkvm/management/models"
)

const claimsKey = "claims"

// EntraClaims are the JWT claims we care about from Entra ID tokens.
type EntraClaims struct {
	OID             string `json:"oid"`
	Email           string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name            string `json:"name"`
}

// GetClaims retrieves the validated Entra claims from the Gin context.
func GetClaims(c *gin.Context) *EntraClaims {
	v, _ := c.Get(claimsKey)
	if v == nil {
		return nil
	}
	cl, _ := v.(*EntraClaims)
	return cl
}

// OIDCAuth returns a Gin middleware that validates Bearer tokens issued by
// Microsoft Entra ID for the given tenant and client audience.
//
// On success it stores the claims under claimsKey and upserts the user record
// in the database so that the first login auto-provisions the account.
func OIDCAuth(tenantID, clientID string, db *sql.DB) gin.HandlerFunc {
	issuer := "https://login.microsoftonline.com/" + tenantID + "/v2.0"

	// Build the OIDC provider (blocking; called once at startup).
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		// Panic here so the operator sees a clear startup error rather than
		// every request returning 401 with a confusing message.
		panic("management: failed to create OIDC provider: " + err.Error())
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return func(c *gin.Context) {
		rawToken := bearerToken(c.GetHeader("Authorization"))
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		idToken, err := verifier.Verify(c.Request.Context(), rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		var claims EntraClaims
		if err := idToken.Claims(&claims); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "malformed claims"})
			return
		}
		if claims.OID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing oid claim"})
			return
		}

		// Use preferred_username as email fallback.
		email := claims.Email
		if email == "" {
			email = claims.PreferredUsername
		}

		// Upsert user record; auto-provision on first login as a regular user.
		now := time.Now().UTC()
		_, err = db.ExecContext(c.Request.Context(), `
			INSERT INTO users (entra_oid, email, display_name, role, last_login_at)
			VALUES (?, ?, ?, 'user', ?)
			ON CONFLICT(entra_oid) DO UPDATE SET
				email         = excluded.email,
				display_name  = excluded.display_name,
				last_login_at = excluded.last_login_at
		`, claims.OID, email, claims.Name, now)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		c.Set(claimsKey, &claims)
		c.Next()
	}
}

// AdminOnly is a Gin middleware that requires the current user to have the
// "admin" role. Must be placed after OIDCAuth in the chain.
func AdminOnly(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := GetClaims(c)
		if cl == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}

		var role string
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT role FROM users WHERE entra_oid = ?`, cl.OID,
		).Scan(&role)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user not found"})
			return
		}
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			return
		}

		c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// UserFromDB loads the full user record (including role) for the given OID.
func UserFromDB(ctx context.Context, db *sql.DB, oid string) (*models.User, error) {
	row := db.QueryRowContext(ctx,
		`SELECT entra_oid, email, display_name, role, last_login_at FROM users WHERE entra_oid = ?`, oid)
	u := &models.User{}
	var lastLogin sql.NullTime
	if err := row.Scan(&u.EntraOID, &u.Email, &u.DisplayName, &u.Role, &lastLogin); err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return u, nil
}
