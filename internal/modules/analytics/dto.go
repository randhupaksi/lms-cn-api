package analytics

type Metric struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type Dashboard struct {
	Role    string   `json:"role"`
	Metrics []Metric `json:"metrics"`
}

type ExamSummary struct {
	ExamID           string         `json:"exam_id"`
	ParticipantCount int64          `json:"participant_count"`
	StartedCount     int64          `json:"started_count"`
	SubmittedCount   int64          `json:"submitted_count"`
	ExpiredCount     int64          `json:"expired_count"`
	AverageScore     float64        `json:"average_score"`
	HighestScore     float64        `json:"highest_score"`
	LowestScore      float64        `json:"lowest_score"`
	AveragePercent   float64        `json:"average_percent"`
	Items            []ItemAnalysis `json:"items"`
}

type ItemAnalysis struct {
	QuestionID    string  `json:"question_id"`
	Stem          string  `json:"stem"`
	AnsweredCount int64   `json:"answered_count"`
	CorrectCount  int64   `json:"correct_count"`
	Accuracy      float64 `json:"accuracy"`
}
