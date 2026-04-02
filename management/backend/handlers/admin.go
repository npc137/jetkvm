package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createDeviceRequest struct {
	Name      string `json:"name"      binding:"required"`
	IPAddress string `json:"ipAddress"  binding:"required"`
	Location  string `json:"location"`
	Password  string `json:"password"`
}

// CreateDevice handles POST /api/admin/devices (admin only).
func CreateDevice(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createDeviceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id := uuid.NewString()
		_, err := db.ExecContext(c.Request.Context(), `
			INSERT INTO devices (id, name, ip_address, location, password)
			VALUES (?, ?, ?, ?, ?)
		`, id, req.Name, req.IPAddress, req.Location, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create device"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "location": req.Location, "status": "unknown"})
	}
}

type updateDeviceRequest struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
	Location  string `json:"location"`
	Password  string `json:"password"`
}

// UpdateDevice handles PUT /api/admin/devices/:id (admin only).
func UpdateDevice(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req updateDeviceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		res, err := db.ExecContext(c.Request.Context(), `
			UPDATE devices
			SET name = COALESCE(NULLIF(?, ''), name),
			    ip_address = COALESCE(NULLIF(?, ''), ip_address),
			    location = COALESCE(NULLIF(?, ''), location),
			    password = COALESCE(NULLIF(?, ''), password)
			WHERE id = ?
		`, req.Name, req.IPAddress, req.Location, req.Password, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// DeleteDevice handles DELETE /api/admin/devices/:id (admin only).
func DeleteDevice(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		res, err := db.ExecContext(c.Request.Context(),
			`DELETE FROM devices WHERE id = ?`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ListUsers handles GET /api/admin/users (admin only).
func ListUsers(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT entra_oid, email, display_name, role, last_login_at
			FROM users ORDER BY display_name
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		defer rows.Close()

		type userRow struct {
			OID         string  `json:"oid"`
			Email       string  `json:"email"`
			DisplayName string  `json:"displayName"`
			Role        string  `json:"role"`
			LastLogin   *string `json:"lastLoginAt,omitempty"`
		}
		var users []userRow
		for rows.Next() {
			var u userRow
			var lastLogin sql.NullString
			if err := rows.Scan(&u.OID, &u.Email, &u.DisplayName, &u.Role, &lastLogin); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
				return
			}
			if lastLogin.Valid {
				u.LastLogin = &lastLogin.String
			}
			users = append(users, u)
		}
		c.JSON(http.StatusOK, users)
	}
}

// SetUserRole handles PUT /api/admin/users/:oid/role (admin only).
func SetUserRole(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		oid := c.Param("oid")
		var body struct {
			Role string `json:"role" binding:"required,oneof=admin user"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		res, err := db.ExecContext(c.Request.Context(),
			`UPDATE users SET role = ? WHERE entra_oid = ?`, body.Role, oid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
