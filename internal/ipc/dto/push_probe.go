package dto

import (
	"time"
	"timon/internal/enums"
)

type PushProbeRequest struct {
	Code    string
	Health  enums.Health
	Comment *string
	Rules   PushProbeRequestRules
}

type PushProbeRequestRules struct {
	Stale         *time.Duration
	StaleIncident *time.Duration
}

type PushProbeResponse struct{}
