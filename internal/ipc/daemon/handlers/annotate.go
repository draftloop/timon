package handlers

import (
	"fmt"
	"github.com/draftloop/elm"
	"time"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
	"timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/dto"
	"timon/internal/validations"
)

func AnnotateHandler(req dto.AnnotateRequest) (res handler.Response[dto.AnnotateResponse]) {
	db := database.GetDB()

	if err := validations.ValidateIncidentAnnotation(req.Note); err != nil {
		return res.SendClientError(err)
	}

	var incident *models.Incident
	if err := db.Model(models.Incident{}).
		Where(elm.Eq("id", req.ID)).
		Scan(&incident); err != nil {
		return res.SendDaemonError(err)
	} else if incident == nil {
		return res.SendClientError(fmt.Errorf("incident does not exist"))
	}

	incidentEvent := models.IncidentEvent{
		Type:       enums.IncidentEventTypeAnnotation,
		IncidentID: incident.ID,
		Note:       req.Note,
		IsSystem:   false,
		At:         time.Now(),
	}
	if err := db.Save(&incidentEvent); err != nil {
		return res.SendDaemonError(err)
	}

	go models.FireWebhookEventIncidentAnnotated(*incident, req.Note)

	return res.Send(dto.AnnotateResponse{})
}
