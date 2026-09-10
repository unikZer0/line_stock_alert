package models

import "time"

type Alert struct {
	ID             string     `json:"id"`
	Symbol         string     `json:"symbol"`
	Condition      string     `json:"condition"`
	TargetPrice    float64    `json:"target_price"`
	Status         string     `json:"status"`
	TriggeredAt    *time.Time `json:"triggered_at,omitempty"`
	DisabledReason *string    `json:"disabled_reason,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AlertFilter struct {
	Symbol string
	Status string
}

type CreateAlertRequest struct {
	Symbol      string  `json:"symbol"`
	Condition   string  `json:"condition"`
	TargetPrice float64 `json:"target_price"`
}

type UpdateAlertRequest struct {
	Condition   string  `json:"condition"`
	TargetPrice float64 `json:"target_price"`
}

type ActiveAlertJob struct {
	ID          string
	UserID      string
	LineUserID  *string
	Symbol      string
	Condition   string
	TargetPrice float64
}
