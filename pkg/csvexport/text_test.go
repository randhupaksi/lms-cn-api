package csvexport

import "testing"

func TestTextPreservesOrdinaryValuesAndQuotesMarkers(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"ordinary name", "Synthetic, Student", "Synthetic, Student"},
		{"empty", "", ""},
		{"blank", "  ", "  "},
		{"leading marker", "+synthetic", "'+synthetic"},
		{"leading spaces and marker", "  @synthetic", "'  @synthetic"},
		{"internal marker", "Synthetic - Student", "Synthetic - Student"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Text(tc.input); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
