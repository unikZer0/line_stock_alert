package models

import (
	"encoding/json"
	"time"
)

type AdminLineMessage struct {
	ID                string     `json:"id"`
	AlertID           *string    `json:"alert_id"`
	UserID            *string    `json:"user_id"`
	Status            string     `json:"status"`
	Message           string     `json:"message"`
	TriggeredPrice    float64    `json:"triggered_price"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	AttemptedAt       time.Time  `json:"attempted_at"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
}

type AdminLineStats struct {
	ConnectedUsers   int64              `json:"connected_users"`
	MessagesSent     int64              `json:"messages_sent"`
	FailedMessages   int64              `json:"failed_messages"`
	DeliveryFailures int64              `json:"delivery_failures"`
	RecentActivity   []AdminLineMessage `json:"recent_activity"`
}

type ApplicationLog struct {
	ID        string          `json:"id"`
	Level     string          `json:"level"`
	Service   string          `json:"service"`
	Code      *string         `json:"code,omitempty"`
	Message   string          `json:"message"`
	Details   json.RawMessage `json:"details,omitempty"`
	RequestID *string         `json:"request_id,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type ApplicationLogFilter struct {
	Level, Service string
	Limit, Offset  int
}
type ApplicationLogPage struct {
	Logs        []ApplicationLog
	Page, Limit int
	Total       int64
}

type AdminAuditLog struct {
	ID          string          `json:"id"`
	AdminUserID *string         `json:"admin_user_id"`
	Action      string          `json:"action"`
	TargetType  string          `json:"target_type"`
	TargetID    *string         `json:"target_id"`
	IPAddress   *string         `json:"ip_address"`
	UserAgent   *string         `json:"user_agent,omitempty"`
	Details     json.RawMessage `json:"details,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type AdminAuditLogPage struct {
	Logs        []AdminAuditLog
	Page, Limit int
	Total       int64
}
