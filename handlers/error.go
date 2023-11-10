package handlers

import (
	"fmt"
	"regexp"
	"strings"

	sqlite3 "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/oSethoum/validator"
)

type ApiResponseError struct {
	MainError error
	Index     int
}

func (err ApiResponseError) Error() string {
	return fmt.Sprintf("error: %s | index: %d", err.MainError.Error(), err.Index)
}

func (e *ApiResponseError) Parse() any {

	errorMap := map[string]any{
		"type":    "other",
		"index":   e.Index,
		"message": e.Error(),
	}

	if mainError, ok := e.MainError.(*validator.Error); ok {
		errorMap["type"] = "validation"
		errorMap["validation"] = mainError.FieldsErrors
	}

	if mainError, ok := e.MainError.(sqlite3.Error); ok {
		errorMap["type"] = "database"

		details := map[string]any{}
		field := regexp.MustCompile(`\w+\.\w+`).FindString(mainError.Error())

		if len(field) > 0 {
			details["field"] = field
		}

		constraint := regexp.MustCompile(`\w+ constraint`).FindString(mainError.Error())

		if len(constraint) > 0 {
			details["constraint"] = strings.Split(constraint, " ")[0]
		}
		errorMap["database"] = details
	}

	return errorMap
}
