package models

type CurrentUser struct {
	ID            string   `json:"id"`
	Email         *string  `json:"email"`
	DisplayName   string   `json:"display_name"`
	EmailVerified bool     `json:"email_verified"`
	Providers     []string `json:"providers"`
	LineConnected bool     `json:"line_connected"`
}
