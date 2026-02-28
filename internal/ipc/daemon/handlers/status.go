package handlers

import (
	"github.com/draftloop/elm"
	"strconv"
	"strings"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
	"timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/dto"
)

func StatusHandler(req dto.StatusRequest) (res handler.Response[dto.StatusResponse]) {
	db := database.GetDB()

	var activeIncidents []models.Incident
	if err := db.Model(models.Incident{}).
		LeftRelation(models.Contract{}).
		Where(elm.NotEq("state", enums.IncidentStateResolved)).
		UnsafeOrderBy("Incident.id ASC").
		Scan(&activeIncidents); err != nil {
		return res.SendDaemonError(err)
	}

	type SelectContracts struct {
		Contract   models.Contract
		LastReport *models.Report
		Incidents  string
	}
	var selectContracts []SelectContracts
	if err := db.Model(models.Contract{}).
		SelectAllFrom(models.Contract{}).
		SelectAllFromAs(models.Report{}, "LastReport").
		UnsafeSelect("(SELECT GROUP_CONCAT(Incident.id) FROM incidents Incident WHERE Incident.contract_id = Contract.id AND Incident.state != \"" + string(enums.IncidentStateResolved) + "\") as Incidents").
		LeftRelation(models.Report{}).
		UnsafeOrderBy("Contract.Type DESC, Contract.Code ASC").
		Scan(&selectContracts); err != nil {
		return res.SendDaemonError(err)
	}

	return res.Send(dto.StatusResponse{
		ActiveIncidents: func() []dto.StatusResponseActiveIncident {
			list := make([]dto.StatusResponseActiveIncident, 0, len(activeIncidents))
			for _, incident := range activeIncidents {
				list = append(list, dto.StatusResponseActiveIncident{
					ID:          incident.ID,
					State:       incident.State,
					TriggerType: incident.TriggerType,
					IsManual:    incident.Contract == nil,
					Title:       incident.Title,
					Description: incident.Description,
					OpenedAt:    incident.OpenedAt,
					RecoveredAt: incident.RecoveredAt,
					RelapsedAt:  incident.RelapsedAt,
					ResolvedAt:  incident.ResolvedAt,
				})
			}
			return list
		}(),
		Contracts: func() []dto.StatusResponseContract {
			list := make([]dto.StatusResponseContract, 0, len(selectContracts))
			for _, row := range selectContracts {
				list = append(list, dto.StatusResponseContract{
					Code:    row.Contract.Code,
					Type:    row.Contract.Type,
					IsStale: row.Contract.IsStale,
					LastReport: func() *dto.StatusResponseContractLastReport {
						if row.LastReport != nil {
							return &dto.StatusResponseContractLastReport{
								UID:             row.LastReport.UID,
								Health:          row.LastReport.Health,
								ProbePushedAt:   row.LastReport.ProbePushedAt,
								ProbeComment:    row.LastReport.ProbeComment,
								JobStartedAt:    row.LastReport.JobStartedAt,
								JobStartComment: row.LastReport.JobStartComment,
								JobEndedAt:      row.LastReport.JobEndedAt,
								JobEndComment:   row.LastReport.JobEndComment,
							}
						}
						return nil
					}(),
					IncidentsID: func() []int64 {
						if row.Incidents == "" {
							return nil
						}
						split := strings.Split(row.Incidents, ",")
						incidentIDs := make([]int64, 0, len(split))
						for _, incidentStr := range split {
							id, err := strconv.ParseInt(incidentStr, 10, 64)
							if err == nil {
								incidentIDs = append(incidentIDs, id)
							}
						}
						return incidentIDs
					}(),
				})
			}
			return list
		}(),
	})
}
