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
	"timon/internal/utils"
	"timon/internal/validations"
)

func ResolveHandler(req dto.ResolveRequest) (res handler.Response[dto.ResolveResponse]) {
	db := database.GetDB()

	if req.Note != nil {
		if err := validations.ValidateIncidentAnnotation(*req.Note); err != nil {
			return res.SendClientError(err)
		}
	}

	var incident *models.Incident
	if err := db.Model(models.Incident{}).
		Where(elm.Eq("id", req.ID)).
		Scan(&incident); err != nil {
		return res.SendDaemonError(err)
	} else if incident == nil {
		return res.SendClientError(fmt.Errorf("incident does not exist"))
	}

	if incident.State == enums.IncidentStateResolved {
		return res.SendClientError(fmt.Errorf("incident is already resolved"))
	}

	incident.State = enums.IncidentStateResolved
	incident.ResolvedAt = utils.Ptr(time.Now())

	if err := db.Save(&incident); err != nil {
		return res.SendDaemonError(err)
	}

	incidentEvent := models.IncidentEvent{
		Type:       enums.IncidentEventTypeIncidentResolved,
		IncidentID: incident.ID,
		Note:       "resolved",
		IsSystem:   false,
		At:         time.Now(),
	}
	if err := db.Save(&incidentEvent); err != nil {
		return res.SendDaemonError(err)
	}

	go models.FireWebhookEventIncidentResolved(*incident, nil)

	if req.Note != nil {
		incidentEvent := models.IncidentEvent{
			Type:       enums.IncidentEventTypeAnnotation,
			IncidentID: incident.ID,
			Note:       *req.Note,
			IsSystem:   false,
			At:         time.Now(),
		}
		if err := db.Save(&incidentEvent); err != nil {
			return res.SendDaemonError(err)
		}

		go models.FireWebhookEventIncidentAnnotated(*incident, *req.Note)
	}

	return res.Send(dto.ResolveResponse{})
}
