package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jetkvm/management/middleware"
)

type createAssignmentRequest struct {
	UserOID    string    `json:"userOid"    binding:"required"`
	DeviceID   string    `json:"deviceId"   binding:"required"`
	ValidFrom  time.Time `json:"validFrom"  binding:"required"`
	ValidUntil time.Time `json:"validUntil" binding:"required"`
}

// CreateAssignment handles POST /api/assignments (admin only).
// Enforces the exclusivity rule: at most one active assignment per user and
// per device at any point in the requested time window.
func CreateAssignment(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createAssignmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !req.ValidUntil.After(req.ValidFrom) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "validUntil must be after validFrom"})
			return
		}

		// Check for conflicting assignment on the same user.
		if conflict, err := hasConflict(c, db, "user_oid", req.UserOID, req.ValidFrom, req.ValidUntil, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		} else if conflict {
			c.JSON(http.StatusConflict, gin.H{"error": "user already has an overlapping assignment"})
			return
		}

		// Check for conflicting assignment on the same device.
		if conflict, err := hasConflict(c, db, "device_id", req.DeviceID, req.ValidFrom, req.ValidUntil, ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		} else if conflict {
			c.JSON(http.StatusConflict, gin.H{"error": "device already has an overlapping assignment"})
			return
		}

		id := uuid.NewString()
		_, err := db.ExecContext(c.Request.Context(), `
			INSERT INTO assignments (id, user_oid, device_id, valid_from, valid_until)
			VALUES (?, ?, ?, ?, ?)
		`, id, req.UserOID, req.DeviceID, req.ValidFrom.UTC(), req.ValidUntil.UTC())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create assignment"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":         id,
			"userOid":    req.UserOID,
			"deviceId":   req.DeviceID,
			"validFrom":  req.ValidFrom,
			"validUntil": req.ValidUntil,
		})
	}
}

// DeleteAssignment handles DELETE /api/assignments/:id (admin only).
func DeleteAssignment(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		res, err := db.ExecContext(c.Request.Context(),
			`DELETE FROM assignments WHERE id = ?`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ListAssignments handles GET /api/assignments (admin only).
func ListAssignments(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT id, user_oid, device_id, valid_from, valid_until, created_at
			FROM assignments ORDER BY valid_from DESC
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		defer rows.Close()

		type row struct {
			ID         string    `json:"id"`
			UserOID    string    `json:"userOid"`
			DeviceID   string    `json:"deviceId"`
			ValidFrom  time.Time `json:"validFrom"`
			ValidUntil time.Time `json:"validUntil"`
			CreatedAt  time.Time `json:"createdAt"`
		}
		var assignments []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.UserOID, &r.DeviceID, &r.ValidFrom, &r.ValidUntil, &r.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
				return
			}
			assignments = append(assignments, r)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		c.JSON(http.StatusOK, assignments)
	}
}

// GetCurrentUser fetches the current user's OID from the Gin context.
func GetCurrentUser(c *gin.Context) string {
	cl := middleware.GetClaims(c)
	if cl == nil {
		return ""
	}
	return cl.OID
}

// hasConflict returns true if there is already an assignment for the given
// column value that overlaps [from, until). excludeID skips one assignment
// (used for update operations).
func hasConflict(c *gin.Context, db *sql.DB, col, val string, from, until time.Time, excludeID string) (bool, error) {
	query := `
		SELECT COUNT(1) FROM assignments
		WHERE ` + col + ` = ?
		  AND id != ?
		  AND valid_from  < ?
		  AND valid_until > ?
	`
	var count int
	err := db.QueryRowContext(c.Request.Context(), query, val, excludeID, until.UTC(), from.UTC()).Scan(&count)
	return count > 0, err
}
