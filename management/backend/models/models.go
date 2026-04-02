package models

import "time"

// User represents an authenticated Entra ID user.
type User struct {
	EntraOID    string     `json:"oid"`
	Email       string     `json:"email"`
	DisplayName string     `json:"displayName"`
	Role        string     `json:"role"` // "admin" or "user"
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

// Device represents a JetKVM unit managed by the plane.
type Device struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	IPAddress   string     `json:"-"` // never exposed to the browser
	Location    string     `json:"location,omitempty"`
	Password    string     `json:"-"` // stored encrypted at rest; never serialised
	LastSeenAt  *time.Time `json:"lastSeenAt,omitempty"`
	Status      string     `json:"status"` // "online" | "offline" | "unknown"
	AssignedTo  *string    `json:"assignedTo,omitempty"` // entra OID of current assignee
	FWVersion   string     `json:"fwVersion,omitempty"`
}

// Assignment maps a user to a device for a date range.
type Assignment struct {
	ID         string    `json:"id"`
	UserOID    string    `json:"userOid"`
	DeviceID   string    `json:"deviceId"`
	ValidFrom  time.Time `json:"validFrom"`
	ValidUntil time.Time `json:"validUntil"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ConnectionToken is the short-lived, single-use token issued when a user
// clicks Connect. It carries the pre-fetched JetKVM authToken so the proxy
// can inject it as a cookie without further device round-trips.
type ConnectionToken struct {
	Token           string     `json:"token"`
	UserOID         string     `json:"userOid"`
	DeviceID        string     `json:"deviceId"`
	DeviceIP        string     `json:"-"`
	JetKVMAuthToken string     `json:"-"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	UsedAt          *time.Time `json:"usedAt,omitempty"`
}
