package governance

import (
	"encoding/json"
	"time"
)

type Settings struct {
	ID               int       `json:"-"`
	SchoolName       string    `json:"school_name"`
	Motto            string    `json:"motto"`
	ContactEmail     string    `json:"contact_email"`
	SupportMessage   string    `json:"support_message"`
	PassingReference float64   `json:"passing_reference"`
	Version          int       `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Settings) TableName() string { return "school_settings" }

type SettingChange struct {
	ID        string          `json:"id"`
	ActorID   string          `json:"-"`
	Version   int             `json:"version"`
	Reason    string          `json:"reason"`
	Snapshot  json.RawMessage `json:"snapshot"`
	CreatedAt time.Time       `json:"created_at"`
}

func (SettingChange) TableName() string { return "school_setting_changes" }

type Event struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"-"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Event) TableName() string { return "school_events" }

type ScheduleRequest struct {
	ID               string     `json:"id"`
	ExamID           string     `json:"exam_id"`
	RequesterID      string     `json:"requester_id"`
	Reason           string     `json:"reason"`
	ProposedStartsAt time.Time  `json:"proposed_starts_at"`
	ProposedEndsAt   time.Time  `json:"proposed_ends_at"`
	OriginalStartsAt time.Time  `json:"original_starts_at"`
	OriginalEndsAt   time.Time  `json:"original_ends_at"`
	Status           string     `json:"status"`
	ReviewerID       *string    `json:"reviewer_id"`
	DecisionNote     string     `json:"decision_note"`
	DecidedAt        *time.Time `json:"decided_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (ScheduleRequest) TableName() string { return "governance_requests" }
