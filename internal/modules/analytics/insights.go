package analytics

import (
	"context"
	"net/http"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/apperror"
)

// Insights are additive to the existing dashboard contract. Dates use school time.
type DashboardInsights struct {
	GeneratedAt       time.Time          `json:"generated_at"`
	Timezone          string             `json:"timezone"`
	StartsOn          string             `json:"starts_on"`
	EndsOn            string             `json:"ends_on"`
	Metrics           []Metric           `json:"metrics"`
	Activity          []ActivityDay      `json:"activity"`
	DistributionTitle string             `json:"distribution_title"`
	Distribution      []DistributionItem `json:"distribution"`
	LearningProgress  *LearningProgress  `json:"learning_progress,omitempty"`
	RecentResults     []RecentResult     `json:"recent_results"`
}
type ActivityDay struct {
	Date                  string `json:"date"`
	ExamSubmissions       int64  `json:"exam_submissions"`
	AssignmentSubmissions int64  `json:"assignment_submissions"`
}
type DistributionItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value int64  `json:"value"`
}
type LearningProgress struct {
	Completed int64 `json:"completed"`
	Total     int64 `json:"total"`
}
type RecentResult struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CourseName  string    `json:"course_name"`
	Percentage  float64   `json:"percentage"`
	PublishedAt time.Time `json:"published_at"`
}
type activityCount struct {
	Day   string
	Total int64
}

var schoolTimezone = time.FixedZone("Asia/Jakarta", 7*60*60)

func insightWindow(now time.Time) (time.Time, []ActivityDay) {
	local := now.In(schoolTimezone)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, schoolTimezone)
	start := today.AddDate(0, 0, -13)
	days := make([]ActivityDay, 14)
	for i := range days {
		days[i].Date = start.AddDate(0, 0, i).Format(time.DateOnly)
	}
	return start.UTC(), days
}

func fillActivity(days []ActivityDay, exams, assignments []activityCount) {
	index := make(map[string]int, len(days))
	for i, day := range days {
		index[day.Date] = i
	}
	for _, row := range exams {
		if i, ok := index[row.Day]; ok {
			days[i].ExamSubmissions = row.Total
		}
	}
	for _, row := range assignments {
		if i, ok := index[row.Day]; ok {
			days[i].AssignmentSubmissions = row.Total
		}
	}
}

func (s *Service) DashboardInsights(ctx context.Context, actor authz.Principal) (DashboardInsights, error) {
	if actor.Role != "admin" && actor.Role != "teacher" && actor.Role != "student" {
		return DashboardInsights{}, apperror.New(http.StatusForbidden, "ANALYTICS_ACCESS_DENIED", "Ringkasan tidak tersedia untuk akun ini")
	}
	now := s.now().UTC()
	start, days := insightWindow(now)
	result := DashboardInsights{GeneratedAt: now, Timezone: "Asia/Jakarta", StartsOn: days[0].Date, EndsOn: days[len(days)-1].Date, Activity: days, RecentResults: []RecentResult{}}
	metrics, err := s.repository.InsightMetrics(ctx, actor.Role, actor.UserID)
	if err != nil {
		return DashboardInsights{}, analyticsReadError(err)
	}
	result.Metrics = metrics
	exams, assignments, err := s.repository.InsightActivity(ctx, actor.Role, actor.UserID, start, now)
	if err != nil {
		return DashboardInsights{}, analyticsReadError(err)
	}
	fillActivity(result.Activity, exams, assignments)
	result.DistributionTitle = "Siklus ujian"
	result.Distribution = []DistributionItem{{Key: "draft", Label: "Draft"}, {Key: "scheduled", Label: "Terjadwal"}, {Key: "ongoing", Label: "Berlangsung"}, {Key: "ended", Label: "Berakhir"}}
	if actor.Role == "student" {
		result.DistributionTitle = "Status tugas aktif"
		result.Distribution = []DistributionItem{{Key: "pending", Label: "Belum dikumpulkan"}, {Key: "overdue", Label: "Lewat tenggat"}, {Key: "submitted", Label: "Menunggu nilai"}, {Key: "graded", Label: "Sudah dinilai"}, {Key: "returned", Label: "Dikembalikan"}}
		progress, err := s.repository.InsightLearningProgress(ctx, actor.UserID)
		if err != nil {
			return DashboardInsights{}, analyticsReadError(err)
		}
		result.LearningProgress = &progress
		result.RecentResults, err = s.repository.InsightRecentResults(ctx, actor.UserID)
		if err != nil {
			return DashboardInsights{}, analyticsReadError(err)
		}
	}
	distribution, err := s.repository.InsightDistribution(ctx, actor.Role, actor.UserID, now)
	if err != nil {
		return DashboardInsights{}, analyticsReadError(err)
	}
	for i := range result.Distribution {
		result.Distribution[i].Value = distribution[result.Distribution[i].Key]
	}
	return result, nil
}
