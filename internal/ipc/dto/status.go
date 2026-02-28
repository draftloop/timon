package dto

import (
	"time"
	"timon/internal/enums"
)

type StatusRequest struct{}

type StatusResponse struct {
	ActiveIncidents []StatusResponseActiveIncident
	Contracts       []StatusResponseContract
}

type StatusResponseActiveIncident struct {
	ID          int64
	State       enums.IncidentState
	TriggerType enums.IncidentTriggerType
	IsManual    bool
	Title       string
	Description *string
	OpenedAt    time.Time
	RecoveredAt *time.Time
	RelapsedAt  *time.Time
	ResolvedAt  *time.Time
}

type StatusResponseContract struct {
	Code        string
	Type        enums.ContractType
	IsStale     bool
	LastReport  *StatusResponseContractLastReport
	IncidentsID []int64
}

type StatusResponseContractLastReport struct {
	UID             string
	Health          enums.Health
	ProbePushedAt   *time.Time
	ProbeComment    *string
	JobStartedAt    *time.Time
	JobStartComment *string
	JobEndedAt      *time.Time
	JobEndComment   *string
}
