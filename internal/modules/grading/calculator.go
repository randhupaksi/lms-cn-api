package grading

type Question struct {
	ID               string
	Type             string
	Points           float64
	CorrectOptionID  string
	SelectedOptionID string
}

type Input struct{ Questions []Question }

type Score struct {
	Earned               float64
	Maximum              float64
	RequiresManualReview bool
}

type Calculator interface{ Calculate(input Input) Score }

type DefaultCalculator struct{}

func (DefaultCalculator) Calculate(input Input) Score {
	var result Score
	for _, question := range input.Questions {
		result.Maximum += question.Points
		switch question.Type {
		case "single_choice":
			if question.SelectedOptionID != "" && question.SelectedOptionID == question.CorrectOptionID {
				result.Earned += question.Points
			}
		default:
			result.RequiresManualReview = true
		}
	}
	return result
}
