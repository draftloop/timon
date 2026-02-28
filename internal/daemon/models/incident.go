package models

import (
	"time"
	"timon/internal/enums"
)

type Incident struct {
	ID          int64
	State       enums.IncidentState
	TriggerType enums.IncidentTriggerType
	Title       string
	Description *string
	ContractID  *int64
	OpenedAt    time.Time
	RecoveredAt *time.Time
	RelapsedAt  *time.Time
	ResolvedAt  *time.Time

	Contract *Contract
}
