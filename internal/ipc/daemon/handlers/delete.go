package handlers

import (
	"fmt"
	"github.com/draftloop/elm"
	"strings"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
	"timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/dto"
	"timon/internal/validations"
)

func DeleteHandler(req dto.DeleteRequest) (res handler.Response[dto.DeleteResponse]) {
	var contractCode string
	var reportUID string
	var err error
	incidentId, _ := validations.ParseIncidentCode(req.Code)
	if incidentId > 0 {
		// ok
	} else if contractCode, reportUID, err = validations.ParseContractCode(req.Code, true); err != nil {
		return res.SendClientError(err)
	}

	db := database.GetDB()

	if incidentId > 0 {
		var incident *models.Incident
		if err := db.Model(models.Incident{}).
			Where(elm.And(
				elm.Eq("Incident.id", incidentId),
			)).
			Scan(&incident); err != nil {
			return res.SendDaemonError(err)
		} else if incident == nil {
			return res.SendClientError(fmt.Errorf("incident does not exist"))
		} else {
			if incident.State != enums.IncidentStateResolved && !req.Force {
				return res.SendClientError(fmt.Errorf("cannot delete INC-%d — the incident is not resolved", incident.ID))
			}
			if err := db.Delete(&incident); err != nil {
				return res.SendDaemonError(err)
			}
		}
	} else if reportUID != "" {
		var report *models.Report
		if err := db.Model(models.Report{}).
			InnerRelation(models.Contract{}).
			Where(elm.And(
				elm.Eq("Contract.code", contractCode),
				elm.Eq("Report.uid", reportUID),
			)).
			Scan(&report); err != nil {
			return res.SendDaemonError(err)
		} else if report == nil {
			return res.SendClientError(fmt.Errorf("sample or run does not exist"))
		} else {
			if !req.Force {
				var incidents []string
				if err := db.Model(models.IncidentEvent{}).
					InnerRelation(models.Incident{}).
					UnsafeSelect("distinct Incident.id").
					Where(elm.And(
						elm.Eq("IncidentEvent.report_id", report.ID),
						elm.NotEq("Incident.state", enums.IncidentStateResolved),
					)).
					Scan(&incidents); err != nil {
					return res.SendDaemonError(err)
				}
				if len(incidents) > 0 {
					return res.SendClientError(fmt.Errorf("cannot delete %q — linked to active incidents: %s — resolve them first", contractCode+":"+reportUID, func() string {
						var v []string
						for _, id := range incidents {
							v = append(v, fmt.Sprintf("INC-%s", id))
						}
						return strings.Join(v, ", ")
					}()))
				}
			}

			contractID := report.ContractID
			if err := db.Delete(&report); err != nil {
				return res.SendDaemonError(err)
			}

			var lastReport *models.Report
			if err := db.Model(models.Report{}).
				Where(elm.And(
					elm.Eq("contract_id", contractID),
				)).
				UnsafeOrderBy("id DESC").
				Scan(&lastReport); err != nil {
				return res.SendDaemonError(err)
			} else if lastReport == nil {
				if err := db.Model(models.Contract{}).
					Set("last_report_id", nil).
					Where(elm.Eq("id", contractID)).
					Update(); err != nil {
					return res.SendDaemonError(err)
				}
			} else {
				if err := db.Model(models.Contract{}).
					Set("last_report_id", lastReport.ID).
					Where(elm.Eq("id", contractID)).
					Update(); err != nil {
					return res.SendDaemonError(err)
				}
			}
		}
	} else {
		var contract *models.Contract
		if err := db.Model(models.Contract{}).
			Where(elm.Eq("code", contractCode)).
			Scan(&contract); err != nil {
			return res.SendDaemonError(err)
		} else if contract == nil {
			return res.SendClientError(fmt.Errorf("probe or job does not exist"))
		} else {
			if !req.Force {
				var incidents []string
				if err := db.Model(models.Incident{}).
					UnsafeSelect("id").
					Where(elm.And(
						elm.Eq("contract_id", contract.ID),
						elm.NotEq("state", enums.IncidentStateResolved),
					)).
					Scan(&incidents); err != nil {
					return res.SendDaemonError(err)
				}
				if len(incidents) > 0 {
					return res.SendClientError(fmt.Errorf("cannot delete %q — linked to active incidents: %s — resolve them first", contractCode, func() string {
						var v []string
						for _, id := range incidents {
							v = append(v, fmt.Sprintf("INC-%s", id))
						}
						return strings.Join(v, ", ")
					}()))
				}
			}

			if err := db.Delete(&contract); err != nil {
				return res.SendDaemonError(err)
			}
		}
	}

	return res.Send(dto.DeleteResponse{})
}
