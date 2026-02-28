package enums

import "fmt"

type Health string

const (
	HealthHealthy  Health = "healthy"
	HealthWarning  Health = "warning"
	HealthCritical Health = "critical"
	HealthUnknown  Health = "unknown" // reserved for the daemon
)

func ParseHealth(s string) (Health, error) {
	switch Health(s) {
	case HealthHealthy, HealthWarning, HealthCritical:
		return Health(s), nil
	}
	return "", fmt.Errorf("invalid health %q — must be one of: healthy, warning, critical", s)
}
