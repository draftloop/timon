package dto

import (
	"time"
	"timon/internal/enums"
)

type ShowRequest struct {
	Code string
}

type ShowResponse struct {
	Incident *ShowResponseIncident
	Contract *ShowResponseContract
	Report   *ShowResponseReport
}

type ShowResponseIncident struct {
	ID           int64
	State        enums.IncidentState
	TriggerType  enums.IncidentTriggerType
	Title        string
	Description  *string
	ContractID   *int64
	ContractCode *string
	OpenedAt     time.Time
	RecoveredAt  *time.Time
	RelapsedAt   *time.Time
	ResolvedAt   *time.Time
	Timeline     []ShowResponseIncidentEvent
}

type ShowResponseIncidentEvent struct {
	Type     enums.IncidentEventType
	ReportID *int64
	Note     string
	IsSystem bool
	At       time.Time
}

type ShowResponseContract struct {
	Code       string
	Type       enums.ContractType
	IsStale    bool
	LastReport *ShowResponseContractReport
	Reports    []ShowResponseContractReport
}

type ShowResponseContractReport struct {
	UID                     string
	Health                  enums.Health
	ProbePushedAt           *time.Time
	ProbeComment            *string
	JobStartedAt            *time.Time
	JobStartComment         *string
	JobEndedAt              *time.Time
	JobEndComment           *string
	RuleStale               *time.Duration
	RuleStaleIncident       *time.Duration
	RuleJobOvertimeIncident *time.Duration
	RuleJobOverlapIncident  *bool
	IncidentsID             []int64
}

type ShowResponseReport struct {
	Code string
	Type enums.ContractType
	ShowResponseContractReport
	Steps []ShowResponseReportJobRunStep
}

type ShowResponseReportJobRunStep struct {
	Label  string
	Health enums.Health
	At     time.Time
}
