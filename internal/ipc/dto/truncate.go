package dto

import "time"

type TruncateRequest struct {
	Code          *string
	Keep          *time.Duration
	KeepHealthy   *time.Duration
	KeepWarning   *time.Duration
	KeepCritical  *time.Duration
	KeepIncidents *time.Duration
}

type TruncateResponse struct{}
