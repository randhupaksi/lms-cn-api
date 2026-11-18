package materials

import (
	"archive/zip"
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path"
	"strings"
	"unicode"

	"lms-cn-api/pkg/apperror"
)

const (
	maxMaterialAttachmentBytes = 8 << 20
	maxMaterialAttachments     = 5
	maxMaterialAttachmentTotal = 20 << 20
	maxImagePixels             = int64(25_000_000)
)

func validateAttachment(fileName string, data []byte) (string, string, error) {
	if len(data) == 0 || len(data) > maxMaterialAttachmentBytes {
		return "", "", apperror.New(http.StatusRequestEntityTooLarge, "ATTACHMENT_SIZE_INVALID", "Ukuran lampiran harus lebih dari 0 dan maksimal 8 MiB")
	}

	name, err := sanitizeAttachmentName(fileName)
	if err != nil {
		return "", "", err
	}

	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg", ".png":
		config, format, decodeErr := image.DecodeConfig(bytes.NewReader(data))
		if decodeErr != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > maxImagePixels {
			return "", "", invalidAttachmentType()
		}
		if strings.EqualFold(path.Ext(name), ".png") {
			if format != "png" {
				return "", "", invalidAttachmentType()
			}
			return name, "image/png", nil
		}
		if format != "jpeg" {
			return "", "", invalidAttachmentType()
		}
		return name, "image/jpeg", nil
	case ".pdf":
		if !hasPDFHeader(data) {
			return "", "", invalidAttachmentType()
		}
		return name, "application/pdf", nil
	case ".docx":
		if !isWordDocument(data) {
			return "", "", invalidAttachmentType()
		}
		return name, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	default:
		return "", "", invalidAttachmentType()
	}
}

func sanitizeAttachmentName(original string) (string, error) {
	name := path.Base(strings.ReplaceAll(original, `\`, "/"))
	var cleaned strings.Builder
	for _, character := range name {
		if !unicode.IsControl(character) {
			cleaned.WriteRune(character)
		}
	}
	name = strings.TrimSpace(cleaned.String())
	if name == "" || name == "." || name == ".." {
		return "", invalidAttachmentType()
	}

	characters := []rune(name)
	if len(characters) > 180 {
		name = string(characters[:180])
	}
	return name, nil
}

func hasPDFHeader(data []byte) bool {
	limit := min(len(data), 1024)
	return bytes.Contains(data[:limit], []byte("%PDF-"))
}

func isWordDocument(data []byte) bool {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}
	contentTypes := false
	document := false
	for _, file := range reader.File {
		switch file.Name {
		case "[Content_Types].xml":
			contentTypes = true
		case "word/document.xml":
			document = true
		}
	}
	return contentTypes && document
}

func invalidAttachmentType() error {
	return apperror.New(http.StatusUnsupportedMediaType, "ATTACHMENT_TYPE_INVALID", "Format lampiran yang didukung: JPG, PNG, PDF, dan DOCX")
}
