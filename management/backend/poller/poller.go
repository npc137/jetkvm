// Package poller implements a background goroutine that periodically polls all
// registered JetKVM devices for their online/offline status.
package poller

import (
	"context"
	"database/sql"
	"time"

	"github.com/jetkvm/management/handlers"
)

const pollInterval = 30 * time.Second

// Start launches the background polling loop. It returns immediately; the
// loop runs until ctx is cancelled.
func Start(ctx context.Context, db *sql.DB) {
	go func() {
		// Run an initial poll immediately on startup, then on the ticker.
		poll(db)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll(db)
			}
		}
	}()
}

// poll queries all registered devices and updates their status.
func poll(db *sql.DB) {
	rows, err := db.Query(`SELECT id, ip_address FROM devices`)
	if err != nil {
		return
	}
	defer rows.Close()

	type device struct{ id, ip string }
	var devices []device
	for rows.Next() {
		var d device
		if err := rows.Scan(&d.id, &d.ip); err == nil {
			devices = append(devices, d)
		}
	}

	for _, d := range devices {
		handlers.PollDevice(db, d.id, d.ip)
	}
}
