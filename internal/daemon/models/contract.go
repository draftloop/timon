package models

import (
	"time"
	"timon/internal/enums"
)

type Contract struct {
	ID                  int64
	Code                string
	Type                enums.ContractType
	IsStale             bool
	LastReportID        *int64
	RuleStale           *time.Duration
	RuleStaleAt         *time.Time
	RuleStaleIncident   *time.Duration
	RuleStaleIncidentAt *time.Time

	LastReport *Report
}
