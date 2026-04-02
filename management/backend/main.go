package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jetkvm/management/db"
	"github.com/jetkvm/management/handlers"
	"github.com/jetkvm/management/middleware"
	"github.com/jetkvm/management/poller"
)

func main() {
	cfg := LoadConfig()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background device-status polling.
	poller.Start(ctx, database)

	// Build the OIDC auth middleware (panics at startup if Entra is unreachable).
	oidcAuth := middleware.OIDCAuth(cfg.EntraTenantID, cfg.EntraClientID, database)
	adminOnly := middleware.AdminOnly(database)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// ── Public ────────────────────────────────────────────────────────────────
	// Token redemption: browser is redirected here after clicking Connect.
	r.GET("/connect/:token", handlers.RedeemToken(database))

	// ── Internal (called by Traefik forward-auth middleware) ──────────────────
	internal := r.Group("/api/internal")
	{
		internal.GET("/token/:token", handlers.ValidateToken(database))
	}

	// ── Authenticated API ─────────────────────────────────────────────────────
	api := r.Group("/api")
	api.Use(oidcAuth)
	{
		api.GET("/me", handlers.GetMe(database))

		// Devices — regular users see only their assigned device.
		api.GET("/devices", handlers.GetDevices(database))
		api.GET("/devices/:id/status", handlers.GetDeviceStatus(database))

		// Connect — users initiate a device session.
		api.POST("/connect/:deviceId", handlers.ConnectRequest(database, cfg.PublicBaseURL))

		// Assignments — admin only.
		adminAssign := api.Group("/assignments")
		adminAssign.Use(adminOnly)
		{
			adminAssign.GET("", handlers.ListAssignments(database))
			adminAssign.POST("", handlers.CreateAssignment(database))
			adminAssign.DELETE("/:id", handlers.DeleteAssignment(database))
		}

		// Admin device / user management.
		admin := api.Group("/admin")
		admin.Use(adminOnly)
		{
			admin.POST("/devices", handlers.CreateDevice(database))
			admin.PUT("/devices/:id", handlers.UpdateDevice(database))
			admin.DELETE("/devices/:id", handlers.DeleteDevice(database))
			admin.GET("/users", handlers.ListUsers(database))
			admin.PUT("/users/:oid/role", handlers.SetUserRole(database))
		}
	}

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("management backend listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
