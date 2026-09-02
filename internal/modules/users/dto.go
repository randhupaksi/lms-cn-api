package users

import "time"

type CreateRequest struct {
	Identifier        string `json:"identifier" binding:"required,min=3,max=64"`
	FullName          string `json:"full_name" binding:"required,min=2,max=160"`
	Role              Role   `json:"role" binding:"required,oneof=teacher student"`
	TemporaryPassword string `json:"temporary_password" binding:"required,min=8,max=128"`
}

type UpdateRequest struct {
	Identifier string `json:"identifier" binding:"omitempty,min=3,max=64"`
	FullName   string `json:"full_name" binding:"omitempty,min=2,max=160"`
	Status     Status `json:"status" binding:"omitempty,oneof=active inactive"`
}

type ResetCredentialRequest struct {
	TemporaryPassword string `json:"temporary_password" binding:"required,min=8,max=128"`
}

type Response struct {
	ID                 string    `json:"id"`
	Identifier         string    `json:"identifier"`
	FullName           string    `json:"full_name"`
	Role               Role      `json:"role"`
	Status             Status    `json:"status"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func ToResponse(user User) Response {
	return Response{
		ID: user.ID, Identifier: user.Identifier, FullName: user.FullName,
		Role: user.Role, Status: user.Status, MustChangePassword: user.MustChangePassword,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}
