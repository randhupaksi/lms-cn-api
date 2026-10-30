package governance

import "time"

type CreateLeadershipActionInput struct {
	Title       string     `json:"title" binding:"required,min=3,max=180"`
	Description string     `json:"description" binding:"max=2000"`
	Category    string     `json:"category" binding:"required,oneof=academic exam data_quality general"`
	Priority    string     `json:"priority" binding:"required,oneof=low normal high"`
	AssigneeID  string     `json:"assignee_id" binding:"omitempty,uuid"`
	DueAt       *time.Time `json:"due_at"`
}

type UpdateLeadershipActionStatusInput struct {
	Status  string `json:"status" binding:"required,oneof=open in_progress blocked completed cancelled"`
	Note    string `json:"note" binding:"required,min=5,max=1000"`
	Version int    `json:"version" binding:"required,min=1"`
}

type LeadershipActionResponse struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Category      string     `json:"category"`
	Priority      string     `json:"priority"`
	Status        string     `json:"status"`
	AssigneeID    *string    `json:"assignee_id"`
	AssigneeName  string     `json:"assignee_name"`
	CreatedBy     string     `json:"created_by"`
	CreatedByName string     `json:"created_by_name"`
	DueAt         *time.Time `json:"due_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	LatestNote    string     `json:"latest_note"`
	Version       int        `json:"version"`
	Overdue       bool       `json:"overdue"`
}

type LeadershipActionAssignee struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type LeadershipActionUpdateResponse struct {
	ID         string    `json:"id"`
	ActorID    string    `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type leadershipActionListRow struct {
	LeadershipAction
	AssigneeName  string `gorm:"column:assignee_name"`
	CreatedByName string `gorm:"column:created_by_name"`
	LatestNote    string `gorm:"column:latest_note"`
}

type leadershipActionUpdateRow struct {
	LeadershipActionUpdate
	ActorName string `gorm:"column:actor_name"`
}
