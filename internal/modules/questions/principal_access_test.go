package questions

import (
	"context"
	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/pagination"
	"testing"
)

func TestPrincipalCannotReadQuestionBank(t *testing.T) {
	service := NewService(nil, nil, nil)
	for _, courseID := range []string{"", "synthetic-course"} {
		if _, _, err := service.List(context.Background(), authz.Principal{Role: "principal"}, courseID, pagination.Request{}, ListFilter{}); err == nil {
			t.Fatal("principal accessed question bank")
		}
	}
}
