package materials

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestValidateAttachmentPreservesExtensionWhenShorteningName(t *testing.T) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}

	name, contentType, err := validateAttachment("folder/"+strings.Repeat("文", 190)+".PNG", buffer.Bytes())
	if err != nil {
		t.Fatalf("valid PNG was rejected: %v", err)
	}
	if utf8.RuneCountInString(name) != maxAttachmentNameRunes || !strings.HasSuffix(name, ".PNG") || contentType != "image/png" {
		t.Fatal("expected a short filename that preserves the PNG extension and content type")
	}
	if _, _, err := validateAttachment("document.jpg", buffer.Bytes()); err == nil {
		t.Fatal("PNG with JPEG extension should fail validation")
	}
}
