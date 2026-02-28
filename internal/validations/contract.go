package validations

import (
	"fmt"
	"regexp"
	"strings"
)

const codePattern = `[a-z0-9][a-z0-9.\-]*[a-z0-9]`

var (
	codeRegex              = regexp.MustCompile(`^` + codePattern + `$`)
	codeWithReportUIDRegex = regexp.MustCompile(`^` + codePattern + `:` + reportUIDPattern + `$`)
)

func ParseContractCode(code string, allowReportUID bool) (string, string, error) {
	if len(code) < 2 || len(code) > 50 {
		return "", "", fmt.Errorf("invalid code %q — must be 2–50 characters", code)
	} else if _, err := ParseIncidentCode(code); err == nil {
		return "", "", fmt.Errorf("invalid code %q — codes matching INC-[0-9]+ are reserved", code)
	}

	invalidMsg := "must be lowercase alphanumeric and may contain '.' or '-' (not at start or end)"
	if allowReportUID {
		invalidMsg += ", optionally followed by a sample or run uid separated by ':' made of 12 lowercase hexadecimal characters"
	}

	switch {
	case codeRegex.MatchString(code):
		return code, "", nil
	case codeWithReportUIDRegex.MatchString(code):
		if !allowReportUID {
			return "", "", fmt.Errorf("invalid code %q — %s, no sample or run uid allowed here", code, invalidMsg)
		}
		parts := strings.SplitN(code, ":", 2)
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid code %q — %s", code, invalidMsg)
	}
}
