package dto

type PushIncidentRequest struct {
	Title       string
	Description *string
}

type PushIncidentResponse struct {
	ID int64
}
