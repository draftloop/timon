package dto

import (
	"time"
	"timon/internal/enums"
)

type PushJobStartRequest struct {
	Code    string
	Comment *string
	Rules   PushJobStartRequestRules
}

type PushJobStartRequestRules struct {
	Stale               *time.Duration
	StaleIncident       *time.Duration
	JobOvertimeIncident *time.Duration
	JobOverlapIncident  *bool
}

type PushJobStartResponse struct {
	RunUID string
}

type PushJobStepRequest struct {
	Code       string
	Label      string
	Health     enums.Health
	RunUID     string
	End        bool
	EndComment *string
}

type PushJobStepResponse struct{}

type PushJobEndRequest struct {
	Code    string
	RunUID  string
	Comment *string
}

type PushJobEndResponse struct{}
