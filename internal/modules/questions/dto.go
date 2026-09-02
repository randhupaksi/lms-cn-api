package questions

import "time"

type OptionRequest struct {
	Content   string `json:"content" binding:"required,max=2000"`
	IsCorrect bool   `json:"is_correct"`
}

type WriteRequest struct {
	CourseID      string          `json:"course_id" binding:"required"`
	Type          string          `json:"type" binding:"required,oneof=single_choice"`
	Stem          string          `json:"stem" binding:"required,max=10000"`
	DefaultPoints float64         `json:"default_points" binding:"required,gt=0,lte=1000"`
	Category      string          `json:"category" binding:"max=80"`
	Tags          []string        `json:"tags" binding:"max=10,dive,max=40"`
	Options       []OptionRequest `json:"options" binding:"required,min=2,max=10,dive"`
}

type ListFilter struct {
	Category string
	Tag      string
	Status   string
	Search   string
}

type OptionResponse struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	IsCorrect bool   `json:"is_correct"`
	Position  uint   `json:"position"`
}

type Response struct {
	ID            string           `json:"id"`
	CourseID      string           `json:"course_id"`
	AuthorID      string           `json:"author_id"`
	Type          string           `json:"type"`
	Stem          string           `json:"stem"`
	DefaultPoints float64          `json:"default_points"`
	Category      string           `json:"category"`
	Tags          []string         `json:"tags"`
	Status        string           `json:"status"`
	Version       uint             `json:"version"`
	Options       []OptionResponse `json:"options"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

func toResponse(question Question) Response {
	options := make([]OptionResponse, len(question.Options))
	for index, option := range question.Options {
		options[index] = OptionResponse{ID: option.ID, Content: option.Content, IsCorrect: option.IsCorrect, Position: option.Position}
	}
	return Response{
		ID: question.ID, CourseID: question.CourseID, AuthorID: question.AuthorID,
		Type: question.Type, Stem: question.Stem, DefaultPoints: question.DefaultPoints,
		Category: question.Category, Tags: question.Tags,
		Status: question.Status, Version: question.Version, Options: options,
		CreatedAt: question.CreatedAt, UpdatedAt: question.UpdatedAt,
	}
}
