package models

import "time"

type AdminUser struct {
	ID             string     `json:"id"`
	Email          *string    `json:"email"`
	DisplayName    string     `json:"display_name"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	EmailVerified  bool       `json:"email_verified"`
	LineConnected  bool       `json:"line_connected"`
	DisabledReason *string    `json:"disabled_reason,omitempty"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AdminUserDetail struct {
	AdminUser
	Providers []string `json:"providers"`
	Alerts    []Alert  `json:"alerts"`
}

type AdminUserFilter struct {
	Search, Status string
	Limit, Offset  int
}

type AdminUserPage struct {
	Users       []AdminUser
	Page, Limit int
	Total       int64
}

type DisableUserRequest struct {
	Reason string `json:"reason"`
}

type AdminActionContext struct{ AdminUserID, IPAddress, UserAgent string }
