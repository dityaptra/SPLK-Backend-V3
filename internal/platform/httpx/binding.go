package httpx

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// FieldError mewakili satu kesalahan validasi pada field tertentu.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// BindJSON mem-bind body request JSON ke struct obj dan memvalidasinya.
// Jika terjadi kesalahan validasi field (validator.ValidationErrors),
// fungsi ini mengembalikan *AppError 400 (Bad Request) dengan detail per field.
// Jika JSON rusak atau malformed, mengembalikan *AppError 400 JSON tidak valid.
func BindJSON(c *gin.Context, obj any) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			details := make([]FieldError, 0, len(valErrs))
			for _, fe := range valErrs {
				details = append(details, FieldError{
					Field:   fe.Field(),
					Tag:     fe.Tag(),
					Message: formatValidationMessage(fe),
				})
			}
			return BadRequest("VALIDASI_GAGAL", "data yang dikirim tidak memenuhi syarat validasi", details)
		}

		return BadRequest("JSON_TIDAK_VALID", fmt.Sprintf("format payload JSON tidak valid atau rusak: %v", err))
	}

	return nil
}

func formatValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("field '%s' wajib diisi", fe.Field())
	case "email":
		return fmt.Sprintf("field '%s' harus berupa alamat email yang valid", fe.Field())
	case "min":
		return fmt.Sprintf("field '%s' minimal berukuran %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("field '%s' maksimal berukuran %s", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("field '%s' harus salah satu dari: %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("field '%s' melanggar aturan '%s'", fe.Field(), fe.Tag())
	}
}
