package validations

import (
	"fmt"
	"regexp"
	"strconv"
)

var incidentCodeRegex = regexp.MustCompile(`^INC-\d+$`)

func ParseIncidentCode(code string) (int64, error) {
	if !incidentCodeRegex.MatchString(code) {
		return 0, fmt.Errorf("invalid incident code %q — must match INC-[0-9]+", code)
	}

	return strconv.ParseInt(code[4:], 10, 64)
}

func ValidateIncidentTitle(title string) error {
	if len(title) < 1 || len(title) > 50 {
		return fmt.Errorf("invalid title — must be 1–50 characters")
	}
	return nil
}

func ValidateIncidentDescription(description string) error {
	if len(description) < 1 || len(description) > 250 {
		return fmt.Errorf("invalid description — must be 1–250 characters")
	}
	return nil
}

func ValidateIncidentAnnotation(note string) error {
	if len(note) < 1 || len(note) > 250 {
		return fmt.Errorf("invalid note — must be 1–250 characters")
	}
	return nil
}
