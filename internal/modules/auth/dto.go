package auth

import "lms-cn-api/internal/modules/users"

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required,min=3,max=64"`
	Password   string `json:"password" binding:"required,min=8,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=8,max=128"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}

type SessionResponse struct {
	AccessToken string         `json:"access_token"`
	ExpiresIn   int64          `json:"expires_in"`
	User        users.Response `json:"user"`
}
