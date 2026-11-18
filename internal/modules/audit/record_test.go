package audit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

type failingRecorder struct{}

func (failingRecorder) Record(context.Context, Record) error {
	return errors.New("database error includes sensitive bound values")
}

func TestRecordBestEffortLogsFailureWithoutUnderlyingError(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	RecordBestEffort(context.Background(), failingRecorder{}, Record{
		Action: "results.published", EntityType: "exam",
	})

	logged := output.String()
	for _, expected := range []string{"audit record failed", "results.published", "exam"} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("expected structured log to contain %q, got %q", expected, logged)
		}
	}
	if strings.Contains(logged, "database error includes sensitive bound values") {
		t.Fatalf("underlying error leaked into log: %q", logged)
	}
}

func TestRecordBestEffortLogsMissingRecorder(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	RecordBestEffort(context.Background(), nil, Record{Action: "exam.published", EntityType: "exam"})

	if !strings.Contains(output.String(), "audit recorder unavailable") {
		t.Fatalf("expected missing recorder to be logged, got %q", output.String())
	}
}
