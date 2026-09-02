package exams

import "time"

type WriteRequest struct {
	CourseID            string    `json:"course_id" binding:"required"`
	Title               string    `json:"title" binding:"required,max=180"`
	Description         string    `json:"description" binding:"max=5000"`
	StartsAt            time.Time `json:"starts_at" binding:"required"`
	EndsAt              time.Time `json:"ends_at" binding:"required"`
	DurationMinutes     uint      `json:"duration_minutes" binding:"required,min=1,max=1440"`
	AllowBackNavigation bool      `json:"allow_back_navigation"`
	RandomizeQuestions  bool      `json:"randomize_questions"`
	RandomizeOptions    bool      `json:"randomize_options"`
}

type QuestionSelection struct {
	QuestionID string  `json:"question_id" binding:"required"`
	Points     float64 `json:"points" binding:"required,gt=0,lte=1000"`
}

type SetQuestionsRequest struct {
	Questions []QuestionSelection `json:"questions" binding:"required,min=1,dive"`
}

type SetParticipantsRequest struct {
	StudentIDs []string `json:"student_ids" binding:"required,min=1,dive,required"`
}

type OptionResponse struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Position uint   `json:"position"`
}

type QuestionResponse struct {
	ID               string           `json:"id"`
	SourceQuestionID string           `json:"source_question_id"`
	Type             string           `json:"type"`
	Stem             string           `json:"stem"`
	Position         uint             `json:"position"`
	Points           float64          `json:"points"`
	Options          []OptionResponse `json:"options"`
}

type Response struct {
	ID                  string             `json:"id"`
	CourseID            string             `json:"course_id"`
	AuthorID            string             `json:"author_id"`
	Title               string             `json:"title"`
	Description         string             `json:"description"`
	Status              string             `json:"status"`
	StartsAt            time.Time          `json:"starts_at"`
	EndsAt              time.Time          `json:"ends_at"`
	DurationMinutes     uint               `json:"duration_minutes"`
	MaxAttempts         uint               `json:"max_attempts"`
	AllowBackNavigation bool               `json:"allow_back_navigation"`
	RandomizeQuestions  bool               `json:"randomize_questions"`
	RandomizeOptions    bool               `json:"randomize_options"`
	ResultPolicy        string             `json:"result_policy"`
	PublishedAt         *time.Time         `json:"published_at"`
	QuestionCount       int                `json:"question_count"`
	ParticipantCount    int64              `json:"participant_count"`
	ParticipantIDs      []string           `json:"participant_ids,omitempty"`
	TotalPoints         float64            `json:"total_points"`
	Questions           []QuestionResponse `json:"questions,omitempty"`
}

func toResponse(exam Exam, participantCount int64, includeQuestions bool) Response {
	result := Response{
		ID: exam.ID, CourseID: exam.CourseID, AuthorID: exam.AuthorID, Title: exam.Title,
		Description: exam.Description, Status: exam.Status, StartsAt: exam.StartsAt, EndsAt: exam.EndsAt,
		DurationMinutes: exam.DurationMinutes, MaxAttempts: exam.MaxAttempts,
		AllowBackNavigation: exam.AllowBackNavigation, ResultPolicy: exam.ResultPolicy,
		RandomizeQuestions: exam.RandomizeQuestions, RandomizeOptions: exam.RandomizeOptions,
		PublishedAt: exam.PublishedAt, QuestionCount: len(exam.Questions), ParticipantCount: participantCount,
	}
	for _, question := range exam.Questions {
		result.TotalPoints += question.Points
	}
	if includeQuestions {
		result.Questions = make([]QuestionResponse, len(exam.Questions))
		for i, question := range exam.Questions {
			options := make([]OptionResponse, len(question.Options))
			for j, option := range question.Options {
				options[j] = OptionResponse{ID: option.ID, Content: option.Content, Position: option.Position}
			}
			result.Questions[i] = QuestionResponse{ID: question.ID, SourceQuestionID: question.SourceQuestionID, Type: question.Type, Stem: question.Stem, Position: question.Position, Points: question.Points, Options: options}
		}
	}
	return result
}
