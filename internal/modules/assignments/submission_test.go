package assignments

import "testing"

func TestValidateSubmissionAttachmentURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "optional empty value", value: "", valid: true},
		{name: "https link", value: "https://example.test/file.pdf", valid: true},
		{name: "http link", value: "http://example.test/file.pdf", valid: true},
		{name: "javascript scheme", value: "javascript:alert(1)"},
		{name: "relative path", value: "/files/document.pdf"},
		{name: "missing host", value: "https:///file.pdf"},
		{name: "embedded credentials", value: "https://student:secret@example.test/file.pdf"},
		{name: "malformed escape", value: "https://example.test/%zz"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSubmissionAttachmentURL(test.value)
			if (err == nil) != test.valid {
				t.Fatalf("validateSubmissionAttachmentURL(%q) error = %v, valid = %v", test.value, err, test.valid)
			}
		})
	}
}

func TestSameSubmissionContent(t *testing.T) {
	base := Submission{Content: "answer", AttachmentURL: "https://example.test/file"}
	if !sameSubmissionContent(base, base) {
		t.Fatal("identical payload should be treated as an idempotent retry")
	}
	if sameSubmissionContent(base, Submission{Content: "changed", AttachmentURL: base.AttachmentURL}) {
		t.Fatal("changed answer should not be treated as an idempotent retry")
	}
	if sameSubmissionContent(base, Submission{Content: base.Content, AttachmentURL: "https://example.test/other"}) {
		t.Fatal("changed attachment should not be treated as an idempotent retry")
	}
}
