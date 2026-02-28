package handlers

import (
	"github.com/draftloop/elm"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
	"timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/dto"
)

func SummaryHandler(req dto.SummaryRequest) (res handler.Response[dto.SummaryResponse]) {
	db := database.GetDB()

	var nbActiveIncidents int
	if err := db.Model(models.Incident{}).
		UnsafeSelect("count(*)").
		Where(elm.NotEq("state", enums.IncidentStateResolved)).
		Scan(&nbActiveIncidents); err != nil {
		return res.SendDaemonError(err)
	}

	var criticalContracts []string
	if err := db.Model(models.Contract{}).
		InnerRelation(models.Report{}).
		UnsafeSelect("code").
		Where(elm.And(
			elm.Eq("health", enums.HealthCritical),
			elm.Eq("is_stale", false),
		)).
		Scan(&criticalContracts); err != nil {
		return res.SendDaemonError(err)
	}

	var staleContracts []string
	if err := db.Model(models.Contract{}).
		LeftRelation(models.Report{}).
		UnsafeSelect("code").
		Where(elm.Eq("is_stale", true)).
		Scan(&staleContracts); err != nil {
		return res.SendDaemonError(err)
	}

	var nbWarningContracts int
	if err := db.Model(models.Contract{}).
		InnerRelation(models.Report{}).
		UnsafeSelect("count(*)").
		Where(elm.And(
			elm.Eq("health", enums.HealthWarning),
			elm.Eq("is_stale", false),
		)).
		Scan(&nbWarningContracts); err != nil {
		return res.SendDaemonError(err)
	}

	var nbHealthyContracts int
	if err := db.Model(models.Contract{}).
		InnerRelation(models.Report{}).
		UnsafeSelect("count(*)").
		Where(elm.And(
			elm.Eq("health", enums.HealthHealthy),
			elm.Eq("is_stale", false),
		)).
		Scan(&nbHealthyContracts); err != nil {
		return res.SendDaemonError(err)
	}

	var nbRunningJobs int
	if err := db.Model(models.Contract{}).
		InnerRelation(models.Report{}).
		UnsafeSelect("count(*)").
		Where(elm.And(
			elm.IsNotNull("job_started_at"),
			elm.IsNull("job_ended_at"),
			elm.Eq("is_stale", false),
		)).
		Scan(&nbRunningJobs); err != nil {
		return res.SendDaemonError(err)
	}

	return res.Send(dto.SummaryResponse{
		ActiveIncidents:    nbActiveIncidents,
		CriticalContracts:  criticalContracts,
		StaleContracts:     staleContracts,
		NbWarningContracts: nbWarningContracts,
		NbHealthyContracts: nbHealthyContracts,
		NbRunningJobs:      nbRunningJobs,
	})
}
