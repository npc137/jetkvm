package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jetkvm/management/middleware"
)

// GetMe handles GET /api/me — returns the current user's profile and role.
func GetMe(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := middleware.GetClaims(c)
		u, err := middleware.UserFromDB(c.Request.Context(), db, cl.OID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load user"})
			return
		}
		c.JSON(http.StatusOK, u)
	}
}
