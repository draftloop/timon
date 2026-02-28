package models

import (
	"time"
	"timon/internal/enums"
)

type IncidentEvent struct {
	ID         int64
	Type       enums.IncidentEventType
	IncidentID int64
	ReportID   *int64
	Note       string
	IsSystem   bool
	At         time.Time

	Incident *Incident
	Report   *Report
}
