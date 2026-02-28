package validations

import (
	"fmt"
	"regexp"
)

const reportUIDPattern = `[a-f0-9]{12}`

var reportUIDRegex = regexp.MustCompile(`^` + reportUIDPattern + `$`)

func ValidateReportUID(uid string) error {
	if reportUIDRegex.MatchString(uid) {
		return nil
	}
	return fmt.Errorf("invalid report uid %q — must be 12 lowercase hexadecimal characters", uid)
}

func ValidateReportComment(comment string) error {
	if len(comment) < 1 || len(comment) > 250 {
		return fmt.Errorf("invalid comment — must be 1–250 characters")
	}
	return nil
}

func ValidateReportJobLabel(label string) error {
	if len(label) < 1 || len(label) > 250 {
		return fmt.Errorf("invalid label %q — must be 1–250 characters", label)
	}
	return nil
}
