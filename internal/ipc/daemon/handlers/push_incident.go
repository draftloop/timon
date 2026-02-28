package handlers

import (
	"time"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
	"timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/dto"
	"timon/internal/validations"
)

func PushIncidentHandler(req dto.PushIncidentRequest) (res handler.Response[dto.PushIncidentResponse]) {
	if err := validations.ValidateIncidentTitle(req.Title); err != nil {
		return res.SendClientError(err)
	}

	if req.Description != nil {
		if err := validations.ValidateIncidentDescription(*req.Description); err != nil {
			return res.SendClientError(err)
		}
	}

	db := database.GetDB()

	incident := models.Incident{
		State:       enums.IncidentStateOpen,
		TriggerType: enums.IncidentTriggerTypeManual,
		Title:       req.Title,
		Description: req.Description,
		OpenedAt:    time.Now(),
	}
	if err := db.Save(&incident); err != nil {
		return res.SendDaemonError(err)
	}

	go models.FireWebhookEventIncidentOpened(incident, nil)

	return res.Send(dto.PushIncidentResponse{
		ID: incident.ID,
	})
}
