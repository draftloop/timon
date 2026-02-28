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

func PushProbeHandler(req dto.PushProbeRequest) (res handler.Response[dto.PushProbeResponse]) {
	if _, _, err := validations.ParseContractCode(req.Code, false); err != nil {
		return res.SendClientError(err)
	}

	if req.Comment != nil {
		if err := validations.ValidateReportComment(*req.Comment); err != nil {
			return res.SendClientError(err)
		}
	}

	if _, err := enums.ParseHealth(string(req.Health)); err != nil {
		return res.SendClientError(err)
	}

	if err := models.ValidateContractRules(enums.ContractTypeProbe, map[enums.ContractRule]*time.Duration{
		enums.ContractRuleStale:         req.Rules.Stale,
		enums.ContractRuleStaleIncident: req.Rules.StaleIncident,
	}); err != nil {
		return res.SendClientError(err)
	}

	db := database.GetDB()

	var contract *models.Contract
	if err := db.Model(models.Contract{}).
		Where(elm.Eq("code", req.Code)).
		Scan(&contract); err != nil {
		return res.SendDaemonError(err)
	} else if contract == nil {
		contract = &models.Contract{
			Code: req.Code,
			Type: enums.ContractTypeProbe,
		}

		if err := db.Save(&contract); err != nil {
			return res.SendDaemonError(err)
		}
	} else {
		if contract.Type != enums.ContractTypeProbe {
			return res.SendClientError(fmt.Errorf("code %q is already used by a job", req.Code))
		}
	}

	oldContractRuleStaleIncident := contract.RuleStaleIncident

	if req.Rules.Stale != nil && req.Rules.StaleIncident != nil && *req.Rules.Stale == *req.Rules.StaleIncident {
		req.Rules.Stale = nil
	}

	report := models.Report{
		UID:               models.GenerateReportUID(),
		ContractID:        contract.ID,
		ProbePushedAt:     utils.Ptr(time.Now()),
		ProbeComment:      req.Comment,
		Health:            req.Health,
		RuleStale:         req.Rules.Stale,
		RuleStaleIncident: req.Rules.StaleIncident,
	}

	contract.RuleStale = req.Rules.Stale
	contract.RuleStaleAt = nil
	if req.Rules.Stale != nil {
		contract.RuleStaleAt = utils.Ptr(report.ProbePushedAt.Add(*req.Rules.Stale))
	}
	contract.IsStale = false

	contract.RuleStaleIncident = req.Rules.StaleIncident
	contract.RuleStaleIncidentAt = nil
	if req.Rules.StaleIncident != nil {
		contract.RuleStaleIncidentAt = utils.Ptr(report.ProbePushedAt.Add(*req.Rules.StaleIncident))
	}

	if err := db.Save(&report); err != nil {
		return res.SendDaemonError(err)
	}

	contract.LastReportID = &report.ID

	if err := db.Save(&contract); err != nil {
		return res.SendDaemonError(err)
	}

	if err := models.ReviseIncidentForReportHealth(enums.ContractTypeProbe, contract.ID, contract.Code, report.ID, report.UID, report.Health, req.Comment, nil); err != nil {
		return res.SendDaemonError(err)
	}

	if oldContractRuleStaleIncident != nil {
		if err := models.ReviseIncidentForStale(contract.ID, contract.Code, *oldContractRuleStaleIncident, false); err != nil {
			return res.SendDaemonError(err)
		}
	}

	return res.Send(dto.PushProbeResponse{})
}
