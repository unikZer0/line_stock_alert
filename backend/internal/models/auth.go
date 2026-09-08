package models

import "time"

type RegisterRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type RegisterResponse struct {
	UserID               string `json:"user_id"`
	Email                string `json:"email"`
	VerificationRequired bool   `json:"verification_required"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type ResendOTPRequest struct {
	Email string `json:"email"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	Role            string
	Status          string
	EmailVerifiedAt *time.Time
}

type VerificationToken struct {
	ID        string
	UserID    string
	OTPHash   string
	ExpiresAt time.Time
	UsedAt    *time.Time
	Attempts  int
	CreatedAt time.Time
}

type RefreshToken struct {
	ID        string
	UserID    string
	FamilyID  string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}
