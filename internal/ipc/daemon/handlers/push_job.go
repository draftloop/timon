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

func PushJobStartHandler(req dto.PushJobStartRequest) (res handler.Response[dto.PushJobStartResponse]) {
	if _, _, err := validations.ParseContractCode(req.Code, false); err != nil {
		return res.SendClientError(err)
	}

	if req.Comment != nil {
		if err := validations.ValidateReportComment(*req.Comment); err != nil {
			return res.SendClientError(err)
		}
	}

	if err := models.ValidateContractRules(enums.ContractTypeJob, map[enums.ContractRule]*time.Duration{
		enums.ContractRuleStale:               req.Rules.Stale,
		enums.ContractRuleStaleIncident:       req.Rules.StaleIncident,
		enums.ContractRuleJobOvertimeIncident: req.Rules.JobOvertimeIncident,
	}); err != nil {
		return res.SendClientError(err)
	}

	db := database.GetDB()

	var contract *models.Contract
	previousReportEnded := true
	if err := db.Model(models.Contract{}).
		Where(elm.Eq("code", req.Code)).
		Scan(&contract); err != nil {
		return res.SendDaemonError(err)
	} else if contract == nil {
		contract = &models.Contract{
			Code: req.Code,
			Type: enums.ContractTypeJob,
		}
		if err := db.Save(&contract); err != nil {
			return res.SendDaemonError(err)
		}
	} else {
		if contract.Type != enums.ContractTypeJob {
			return res.SendClientError(fmt.Errorf("code %q is already used by a probe", req.Code))
		}

		if contract.LastReportID != nil {
			var previousReportEndedAt *time.Time
			if err := db.Model(models.Report{}).
				UnsafeSelect("job_ended_at").
				Where(elm.Eq("id", *contract.LastReportID)).
				Scan(&previousReportEndedAt); err != nil {
				return res.SendDaemonError(err)
			}

			previousReportEnded = previousReportEndedAt != nil
		}
	}

	oldContractRuleStaleIncident := contract.RuleStaleIncident

	if req.Rules.Stale != nil && req.Rules.StaleIncident != nil && *req.Rules.Stale == *req.Rules.StaleIncident {
		req.Rules.Stale = nil
	}

	report := models.Report{
		UID:                     models.GenerateReportUID(),
		ContractID:              contract.ID,
		JobStartComment:         req.Comment,
		JobStartedAt:            utils.Ptr(time.Now()),
		Health:                  enums.HealthUnknown,
		RuleStale:               req.Rules.Stale,
		RuleStaleIncident:       req.Rules.StaleIncident,
		RuleJobOvertimeIncident: req.Rules.JobOvertimeIncident,
		RuleJobOverlapIncident:  req.Rules.JobOverlapIncident,
	}

	contract.RuleStale = req.Rules.Stale
	contract.RuleStaleAt = nil
	if req.Rules.Stale != nil {
		contract.RuleStaleAt = utils.Ptr(report.JobStartedAt.Add(*req.Rules.Stale))
	}
	contract.IsStale = false

	contract.RuleStaleIncident = req.Rules.StaleIncident
	contract.RuleStaleIncidentAt = nil
	if req.Rules.StaleIncident != nil {
		contract.RuleStaleIncidentAt = utils.Ptr(report.JobStartedAt.Add(*req.Rules.StaleIncident))
	}

	if req.Rules.JobOvertimeIncident != nil {
		report.RuleJobOvertimeIncidentAt = utils.Ptr(report.JobStartedAt.Add(*req.Rules.JobOvertimeIncident))
	}

	if err := db.Save(&report); err != nil {
		return res.SendDaemonError(err)
	}

	contract.LastReportID = &report.ID

	if err := db.Save(&contract); err != nil {
		return res.SendDaemonError(err)
	}

	if req.Rules.JobOverlapIncident != nil && *req.Rules.JobOverlapIncident {
		if err := models.ReviseIncidentForJobOverlap(contract.ID, contract.Code, report.ID, report.UID, previousReportEnded); err != nil {
			return res.SendDaemonError(err)
		}
	}

	if oldContractRuleStaleIncident != nil {
		if err := models.ReviseIncidentForStale(contract.ID, contract.Code, *oldContractRuleStaleIncident, false); err != nil {
			return res.SendDaemonError(err)
		}
	}

	return res.Send(dto.PushJobStartResponse{RunUID: report.UID})
}

func PushJobStepHandler(req dto.PushJobStepRequest) (res handler.Response[dto.PushJobStepResponse]) {
	if _, _, err := validations.ParseContractCode(req.Code, false); err != nil {
		return res.SendClientError(err)
	}

	if err := validations.ValidateReportJobLabel(req.Label); err != nil {
		return res.SendClientError(err)
	}

	if _, err := enums.ParseHealth(string(req.Health)); err != nil {
		return res.SendClientError(err)
	}

	if err := validations.ValidateReportUID(req.RunUID); err != nil {
		return res.SendClientError(err)
	}

	if req.End && req.EndComment != nil {
		if err := validations.ValidateReportComment(*req.EndComment); err != nil {
			return res.SendClientError(err)
		}
	}

	db := database.GetDB()

	var report *models.Report
	if err := db.Model(models.Report{}).
		InnerRelation(models.Contract{}).
		Where(elm.And(
			elm.Eq("Contract.type", enums.ContractTypeJob),
			elm.Eq("Contract.code", req.Code),
			elm.Eq("Report.uid", req.RunUID),
		)).
		Scan(&report); err != nil {
		return res.SendDaemonError(err)
	} else if report == nil {
		return res.SendClientError(fmt.Errorf("run does not exist"))
	} else if report.JobEndedAt != nil {
		return res.SendClientError(fmt.Errorf("job has already ended"))
	}

	jobStep := models.ReportJobStep{
		ReportID: report.ID,
		Label:    req.Label,
		Health:   req.Health,
		At:       time.Now(),
	}

	if req.Health == enums.HealthWarning && report.Health != enums.HealthCritical {
		report.Health = enums.HealthWarning
	} else if req.Health == enums.HealthCritical {
		report.Health = enums.HealthCritical
	} else if req.End && req.Health == enums.HealthHealthy && report.Health == enums.HealthUnknown {
		report.Health = enums.HealthHealthy
	}

	if req.End {
		report.JobEndedAt = &jobStep.At
		if req.EndComment != nil {
			report.JobEndComment = req.EndComment
		} else {
			report.JobEndComment = &req.Label
		}
	}

	if err := db.Save(&report); err != nil {
		return res.SendDaemonError(err)
	}

	if err := db.Save(&jobStep); err != nil {
		return res.SendDaemonError(err)
	}

	if req.End {
		if err := models.ReviseIncidentForReportHealth(enums.ContractTypeJob, report.Contract.ID, report.Contract.Code, report.ID, report.UID, report.Health, req.EndComment, &req.Label); err != nil {
			return res.SendDaemonError(err)
		}

		if report.RuleJobOvertimeIncident != nil {
			if err := models.ReviseIncidentForJobOvertime(report.Contract.ID, report.Contract.Code, report.ID, report.UID, *report.RuleJobOvertimeIncident, report.JobEndedAt.Sub(*report.JobStartedAt)); err != nil {
				return res.SendDaemonError(err)
			}
		}
	}

	return res.Send(dto.PushJobStepResponse{})
}

func PushJobEndHandler(req dto.PushJobEndRequest) (res handler.Response[dto.PushJobEndResponse]) {
	if _, _, err := validations.ParseContractCode(req.Code, false); err != nil {
		return res.SendClientError(err)
	}

	if err := validations.ValidateReportUID(req.RunUID); err != nil {
		return res.SendClientError(err)
	}

	if req.Comment != nil {
		if err := validations.ValidateReportComment(*req.Comment); err != nil {
			return res.SendClientError(err)
		}
	}

	db := database.GetDB()

	var report *models.Report
	if err := db.Model(models.Report{}).
		LeftRelation(models.Contract{}).
		Where(elm.And(
			elm.Eq("Contract.type", enums.ContractTypeJob),
			elm.Eq("Contract.code", req.Code),
			elm.Eq("Report.uid", req.RunUID),
		)).
		Scan(&report); err != nil {
		return res.SendDaemonError(err)
	} else if report == nil {
		return res.SendClientError(fmt.Errorf("run does not exist"))
	} else if report.JobEndedAt != nil {
		return res.SendClientError(fmt.Errorf("job has already ended"))
	}

	var firstFailedStepLabel *string
	if err := db.Model(models.ReportJobStep{}).
		UnsafeSelect("label").
		Where(elm.And(
			elm.Eq("report_id", report.ID),
			elm.Eq("health", enums.HealthCritical),
		)).
		UnsafeOrderBy("at ASC").
		Limit(1).
		Scan(&firstFailedStepLabel); err != nil {
		return res.SendDaemonError(err)
	}

	if report.Health == enums.HealthUnknown {
		report.Health = enums.HealthHealthy
	}

	report.JobEndedAt = utils.Ptr(time.Now())
	report.JobEndComment = req.Comment

	if err := db.Save(&report); err != nil {
		return res.SendDaemonError(err)
	}

	if err := models.ReviseIncidentForReportHealth(enums.ContractTypeJob, report.Contract.ID, report.Contract.Code, report.ID, report.UID, report.Health, req.Comment, firstFailedStepLabel); err != nil {
		return res.SendDaemonError(err)
	}

	if report.RuleJobOvertimeIncident != nil {
		if err := models.ReviseIncidentForJobOvertime(report.Contract.ID, report.Contract.Code, report.ID, report.UID, *report.RuleJobOvertimeIncident, report.JobEndedAt.Sub(*report.JobStartedAt)); err != nil {
			return res.SendDaemonError(err)
		}
	}

	return res.Send(dto.PushJobEndResponse{})
}
