package materials

import "time"

type WriteRequest struct {
	CourseID    string `json:"course_id" binding:"required"`
	Title       string `json:"title" binding:"required,max=180"`
	Description string `json:"description" binding:"max=5000"`
	Content     string `json:"content" binding:"required,max=50000"`
	Position    uint   `json:"position" binding:"required,min=1,max=1000"`
}

type Response struct {
	ID          string     `json:"id"`
	CourseID    string     `json:"course_id"`
	AuthorID    string     `json:"author_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	Position    uint       `json:"position"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type materialRow struct {
	Material
	CompletedAt *time.Time
}

func toResponse(row materialRow) Response {
	return Response{ID: row.ID, CourseID: row.CourseID, AuthorID: row.AuthorID, Title: row.Title, Description: row.Description, Content: row.Content, Position: row.Position, Status: row.Status, PublishedAt: row.PublishedAt, CompletedAt: row.CompletedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
