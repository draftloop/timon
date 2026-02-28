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

func TruncateHandler(req dto.TruncateRequest) (res handler.Response[dto.TruncateResponse]) {
	db := database.GetDB()

	if req.Keep == nil && req.KeepHealthy == nil && req.KeepWarning == nil && req.KeepCritical == nil && req.KeepIncidents == nil {
		return res.SendClientError(fmt.Errorf("at least one keep flag is required"))
	} else if req.Keep != nil && (req.KeepHealthy != nil || req.KeepWarning != nil || req.KeepCritical != nil) {
		return res.SendClientError(fmt.Errorf("keep cannot be combined with keep-healthy, keep-warning, or keep-critical"))
	}

	var contract *models.Contract
	if req.Code != nil {
		if _, _, err := validations.ParseContractCode(*req.Code, false); err != nil {
			return res.SendClientError(err)
		}

		if err := db.Model(models.Contract{}).
			Where(elm.Eq("code", *req.Code)).
			Scan(&contract); err != nil {
			return res.SendDaemonError(err)
		} else if contract == nil {
			return res.SendClientError(fmt.Errorf("probe or job does not exist"))
		}
	}

	if req.KeepIncidents != nil {
		truncation := db.Model(models.Incident{}).
			Where(elm.And(
				elm.Eq("state", enums.IncidentStateResolved),
				elm.LtEq("resolved_at", time.Now().Add(-*req.KeepIncidents)),
			))

		if contract != nil {
			truncation.Where(elm.Eq("contract_id", contract.ID))
		}

		if err := truncation.Delete(); err != nil {
			return res.SendDaemonError(err)
		}
	}

	if req.Keep != nil || req.KeepHealthy != nil || req.KeepWarning != nil || req.KeepCritical != nil {
		truncation := db.Model(models.Report{})

		if req.Keep != nil {
			t := time.Now().Add(-*req.Keep)
			truncation.Where(elm.Or(
				elm.LtEq("probe_pushed_at", t),
				elm.LtEq("COALESCE(job_ended_at, job_started_at)", t),
			))
		} else {
			var or []elm.BuilderWhere
			if req.KeepHealthy != nil {
				t := time.Now().Add(-*req.KeepHealthy)
				or = append(or, elm.And(
					elm.Eq("health", enums.HealthHealthy),
					elm.Or(
						elm.LtEq("probe_pushed_at", t),
						elm.LtEq("job_ended_at", t),
					),
				))
			}
			if req.KeepWarning != nil {
				t := time.Now().Add(-*req.KeepWarning)
				or = append(or, elm.And(
					elm.Eq("health", enums.HealthWarning),
					elm.Or(
						elm.LtEq("probe_pushed_at", t),
						elm.LtEq("job_ended_at", t),
					),
				))
			}
			if req.KeepCritical != nil {
				t := time.Now().Add(-*req.KeepCritical)
				or = append(or, elm.And(
					elm.Eq("health", enums.HealthCritical),
					elm.Or(
						elm.LtEq("probe_pushed_at", t),
						elm.LtEq("job_ended_at", t),
					),
				))
				or = append(or, elm.And(
					elm.IsNotNull("job_started_at"),
					elm.IsNull("job_ended_at"),
					elm.LtEq("job_started_at", t),
				))
			}
			truncation.Where(elm.Or(or...))
		}

		if contract != nil {
			truncation.Where(elm.Eq("contract_id", contract.ID))
		}

		truncation.Where(elm.UnsafeWhere("NOT EXISTS (SELECT 1 FROM incident_events INNER JOIN incidents ON incidents.id = incident_events.incident_id WHERE incident_events.report_id = reports.id AND incidents.state != 'resolved')"))

		if err := truncation.Delete(); err != nil {
			return res.SendDaemonError(err)
		}
	}

	return res.Send(dto.TruncateResponse{})
}
