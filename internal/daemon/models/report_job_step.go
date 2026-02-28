package models

import (
	"time"
	"timon/internal/enums"
)

type ReportJobStep struct {
	ID       int64
	ReportID int64
	Label    string
	Health   enums.Health
	At       time.Time

	Report *Report
}
