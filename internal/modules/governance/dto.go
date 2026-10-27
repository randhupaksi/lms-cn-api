package governance

import "time"

type SettingsInput struct {
	SchoolName       string  `json:"school_name" binding:"required,min=2,max=160"`
	Motto            string  `json:"motto" binding:"max=255"`
	ContactEmail     string  `json:"contact_email" binding:"omitempty,email,max=160"`
	SupportMessage   string  `json:"support_message" binding:"max=1000"`
	PassingReference float64 `json:"passing_reference" binding:"gte=0,lte=100"`
	Version          int     `json:"version" binding:"required,min=1"`
	Reason           string  `json:"reason" binding:"required,min=10,max=500"`
}
type EventInput struct {
	Title       string    `json:"title" binding:"required,min=3,max=180"`
	Description string    `json:"description" binding:"max=2000"`
	Category    string    `json:"category" binding:"required,oneof=academic holiday meeting announcement"`
	StartsAt    time.Time `json:"starts_at" binding:"required"`
	EndsAt      time.Time `json:"ends_at" binding:"required"`
	Version     int       `json:"version" binding:"min=0"`
}
type ScheduleInput struct {
	ExamID           string    `json:"exam_id" binding:"required,uuid"`
	Reason           string    `json:"reason" binding:"required,min=10,max=1000"`
	ProposedStartsAt time.Time `json:"proposed_starts_at" binding:"required"`
	ProposedEndsAt   time.Time `json:"proposed_ends_at" binding:"required"`
}
type DecisionInput struct {
	Decision string `json:"decision" binding:"required,oneof=approved rejected"`
	Note     string `json:"note" binding:"required,min=10,max=1000"`
}
type ExamReadiness struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	CourseID             string    `json:"course_id"`
	CourseName           string    `json:"course_name"`
	ClassName            string    `json:"class_name"`
	Status               string    `json:"status"`
	StartsAt             time.Time `json:"starts_at"`
	EndsAt               time.Time `json:"ends_at"`
	DurationMinutes      int       `json:"duration_minutes"`
	QuestionCount        int64     `json:"question_count"`
	ParticipantCount     int64     `json:"participant_count"`
	InactiveParticipants int64     `json:"inactive_participants"`
	TeacherCount         int64     `json:"teacher_count"`
	AttemptCount         int64     `json:"attempt_count"`
	ConflictCount        int64     `json:"conflict_count"`
	Issues               []string  `json:"issues" gorm:"-"`
	Lifecycle            string    `json:"lifecycle" gorm:"-"`
}
type RoleCount struct {
	Role     string `json:"role"`
	Active   int64  `json:"active"`
	Inactive int64  `json:"inactive"`
}
type Overview struct {
	GeneratedAt     time.Time       `json:"generated_at"`
	Roles           []RoleCount     `json:"roles" gorm:"-"`
	Courses         int64           `json:"courses"`
	Classes         int64           `json:"classes"`
	ActiveAttempts  int64           `json:"active_attempts"`
	PendingRequests int64           `json:"pending_requests"`
	DraftResults    int64           `json:"draft_results"`
	Exams           []ExamReadiness `json:"exams" gorm:"-"`
	ExamStates      []StateCount    `json:"exam_states" gorm:"-"`
	Quality         []QualityGroup  `json:"quality" gorm:"-"`
	Agenda          []Event         `json:"agenda" gorm:"-"`
}
type Finding struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type QualityGroup struct {
	Key         string    `json:"key"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Count       int64     `json:"count"`
	Samples     []Finding `json:"samples"`
}
type RequestRow struct {
	ScheduleRequest
	ExamTitle     string `json:"exam_title"`
	CourseName    string `json:"course_name"`
	RequesterName string `json:"requester_name"`
	ReviewerName  string `json:"reviewer_name"`
}
type CourseReport struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	ClassName         string   `json:"class_name"`
	YearName          string   `json:"year_name"`
	Students          int64    `json:"students"`
	Teachers          int64    `json:"teachers"`
	Exams             int64    `json:"exams"`
	SubmittedAttempts int64    `json:"submitted_attempts"`
	PublishedResults  int64    `json:"published_results"`
	AverageScore      *float64 `json:"average_score"`
	BelowReference    int64    `json:"below_reference"`
	PassingReference  float64  `json:"passing_reference"`
	Assignments       int64    `json:"assignments"`
	PendingGrading    int64    `json:"pending_grading"`
}
