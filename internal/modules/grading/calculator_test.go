package grading

import "testing"

func TestDefaultCalculatorSingleChoice(t *testing.T) {
	calculator := DefaultCalculator{}
	result := calculator.Calculate(Input{Questions: []Question{
		{ID: "q1", Type: "single_choice", Points: 2, CorrectOptionID: "a", SelectedOptionID: "a"},
		{ID: "q2", Type: "single_choice", Points: 3, CorrectOptionID: "b", SelectedOptionID: "c"},
		{ID: "q3", Type: "single_choice", Points: 5, CorrectOptionID: "d"},
	}})
	if result.Earned != 2 || result.Maximum != 10 {
		t.Fatalf("unexpected score: %+v", result)
	}
	if result.RequiresManualReview {
		t.Fatal("single choice should not require manual review")
	}
}

func TestDefaultCalculatorMarksUnknownTypeForManualReview(t *testing.T) {
	result := (DefaultCalculator{}).Calculate(Input{Questions: []Question{{ID: "q1", Type: "essay", Points: 10}}})
	if !result.RequiresManualReview {
		t.Fatal("unknown type must require manual review")
	}
}
