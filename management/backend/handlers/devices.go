package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jetkvm/management/middleware"
	"github.com/jetkvm/management/models"
)

// GetDevices handles GET /api/devices.
//   - Admins receive the full device list.
//   - Regular users receive only the device currently assigned to them.
//
// IP addresses and passwords are never included in the response.
func GetDevices(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := middleware.GetClaims(c)
		u, err := middleware.UserFromDB(c.Request.Context(), db, cl.OID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load user"})
			return
		}

		var devices []models.Device
		if u.Role == "admin" {
			devices, err = listAllDevices(c, db)
		} else {
			devices, err = listUserDevices(c, db, u.EntraOID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list devices"})
			return
		}

		c.JSON(http.StatusOK, devices)
	}
}

// GetDeviceStatus handles GET /api/devices/:id/status — returns the cached
// status for a single device (no live poll; the poller updates periodically).
func GetDeviceStatus(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var status, fwVersion string
		var lastSeen sql.NullTime
		err := db.QueryRowContext(c.Request.Context(),
			`SELECT status, last_seen_at, fw_version FROM devices WHERE id = ?`, id,
		).Scan(&status, &lastSeen, &fwVersion)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		resp := gin.H{"status": status, "fwVersion": fwVersion}
		if lastSeen.Valid {
			resp["lastSeenAt"] = lastSeen.Time
		}
		c.JSON(http.StatusOK, resp)
	}
}

// listAllDevices returns all devices with their current active assignee OID.
func listAllDevices(c *gin.Context, db *sql.DB) ([]models.Device, error) {
	rows, err := db.QueryContext(c.Request.Context(), `
		SELECT d.id, d.name, d.location, d.last_seen_at, d.fw_version, d.status,
		       a.user_oid
		FROM devices d
		LEFT JOIN assignments a
		       ON a.device_id = d.id
		      AND datetime('now') BETWEEN datetime(a.valid_from) AND datetime(a.valid_until)
		ORDER BY d.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeviceRows(rows)
}

// listUserDevices returns only the device currently assigned to the given user.
func listUserDevices(c *gin.Context, db *sql.DB, userOID string) ([]models.Device, error) {
	rows, err := db.QueryContext(c.Request.Context(), `
		SELECT d.id, d.name, d.location, d.last_seen_at, d.fw_version, d.status,
		       a.user_oid
		FROM devices d
		JOIN assignments a
		  ON a.device_id = d.id
		 AND a.user_oid  = ?
		 AND datetime('now') BETWEEN datetime(a.valid_from) AND datetime(a.valid_until)
		ORDER BY d.name
	`, userOID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeviceRows(rows)
}

func scanDeviceRows(rows *sql.Rows) ([]models.Device, error) {
	var devices []models.Device
	for rows.Next() {
		var d models.Device
		var lastSeen sql.NullTime
		var assignedTo sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Location, &lastSeen, &d.FWVersion, &d.Status, &assignedTo); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			d.LastSeenAt = &lastSeen.Time
		}
		if assignedTo.Valid {
			d.AssignedTo = &assignedTo.String
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}
