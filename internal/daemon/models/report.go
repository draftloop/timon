package models

import (
	"fmt"
	"github.com/draftloop/elm"
	"github.com/google/uuid"
	"slices"
	"strings"
	"time"
	database "timon/internal/daemon/db"
	"timon/internal/enums"
	"timon/internal/log"
	"timon/internal/utils"
)

type Report struct {
	ID                        int64
	UID                       string
	ContractID                int64
	Health                    enums.Health
	ProbePushedAt             *time.Time
	ProbeComment              *string
	JobStartedAt              *time.Time
	JobStartComment           *string
	JobEndedAt                *time.Time
	JobEndComment             *string
	RuleStale                 *time.Duration
	RuleStaleIncident         *time.Duration
	RuleJobOvertimeIncident   *time.Duration
	RuleJobOvertimeIncidentAt *time.Time
	RuleJobOverlapIncident    *bool

	Contract *Contract
}

func ReviseIncidentForReportHealth(contractType enums.ContractType, contractID int64, contractCode string, reportID int64, reportUID string, reportHealth enums.Health, comment *string, jobStepLabel *string) error {
	db := database.GetDB()

	reportRef := "run"
	if contractType == enums.ContractTypeJob {
		reportRef = "sample"
	}

	var incident *Incident
	if err := db.Model(Incident{}).
		Where(elm.And(
			elm.NotEq("state", enums.IncidentStateResolved),
			elm.Eq("trigger_type", enums.IncidentTriggerTypeCritical),
			elm.Eq("contract_id", contractID),
		)).
		Scan(&incident); err != nil {
		return err
	}

	if reportHealth == enums.HealthCritical {
		if incident == nil {
			incident = &Incident{
				State:       enums.IncidentStateOpen,
				TriggerType: enums.IncidentTriggerTypeCritical,
				Title:       fmt.Sprintf("%s is critical", contractCode),
				ContractID:  &contractID,
				OpenedAt:    time.Now(),
			}
			if err := db.Save(&incident); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentOpened,
				IncidentID: incident.ID,
				ReportID:   &reportID,
				Note: fmt.Sprintf("%s turned critical (%s %s)%s", contractCode, reportRef, reportUID, func() string {
					if comment != nil {
						return fmt.Sprintf(": %s", *comment)
					}
					if jobStepLabel != nil {
						return fmt.Sprintf(" at step: %s", *jobStepLabel)
					}
					return ""
				}()),
				IsSystem: true,
				At:       time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentOpened(*incident, &incidentEvent.Note)
		} else if incident.State == enums.IncidentStateRecovered {
			if err := db.Model(Incident{}).
				Where(elm.Eq("id", incident.ID)).
				Set("state", enums.IncidentStateRelapsed).
				Set("relapsed_at", time.Now()).
				Update(); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentRelapsed,
				IncidentID: incident.ID,
				ReportID:   &reportID,
				Note: fmt.Sprintf("%s turned critical again (%s %s)%s", contractCode, reportRef, reportUID, func() string {
					if comment != nil {
						return fmt.Sprintf(": %s", *comment)
					}
					if jobStepLabel != nil {
						return fmt.Sprintf(" at step: %s", *jobStepLabel)
					}
					return ""
				}()),
				IsSystem: true,
				At:       time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentRelapsed(*incident, incidentEvent.Note)
		}
	} else if incident != nil && reportHealth == enums.HealthHealthy && incident.State == enums.IncidentStateOpen {
		if err := db.Model(Incident{}).
			Where(elm.Eq("id", incident.ID)).
			Set("state", enums.IncidentStateRecovered).
			Set("recovered_at", time.Now()).
			Update(); err != nil {
			return err
		}
		incident.State = enums.IncidentStateRecovered
		incident.RecoveredAt = utils.Ptr(time.Now())

		incidentEvent := IncidentEvent{
			Type:       enums.IncidentEventTypeIncidentRecovered,
			IncidentID: incident.ID,
			ReportID:   &reportID,
			Note: fmt.Sprintf("%s back to %s (%s %s)%s", contractCode, reportHealth, reportRef, reportUID, func() string {
				if comment != nil {
					return fmt.Sprintf(": %s", *comment)
				}
				return ""
			}()),
			IsSystem: true,
			At:       time.Now(),
		}
		if err := db.Save(&incidentEvent); err != nil {
			return err
		}

		go FireWebhookEventIncidentRecovered(*incident, incidentEvent.Note)
	}

	return nil
}

func ReviseIncidentForJobOverlap(contractID int64, contractCode string, reportID int64, reportUID string, previousReportEnded bool) error {
	db := database.GetDB()

	var incident *Incident
	if err := db.Model(Incident{}).
		Where(elm.And(
			elm.NotEq("state", enums.IncidentStateResolved),
			elm.Eq("trigger_type", enums.IncidentTriggerTypeJobOverlap),
			elm.Eq("contract_id", contractID),
		)).
		Scan(&incident); err != nil {
		return err
	}

	if !previousReportEnded {
		if incident == nil {
			incident = &Incident{
				State:       enums.IncidentStateOpen,
				TriggerType: enums.IncidentTriggerTypeJobOverlap,
				Title:       fmt.Sprintf("%s is overlapping", contractCode),
				ContractID:  &contractID,
				OpenedAt:    time.Now(),
			}
			if err := db.Save(&incident); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentOpened,
				IncidentID: incident.ID,
				ReportID:   &reportID,
				Note:       fmt.Sprintf("%s started overlapping (run %s)", contractCode, reportUID),
				IsSystem:   true,
				At:         time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentOpened(*incident, &incidentEvent.Note)
		} else if incident.State == enums.IncidentStateRecovered {
			if err := db.Model(Incident{}).
				Where(elm.Eq("id", incident.ID)).
				Set("state", enums.IncidentStateRelapsed).
				Set("relapsed_at", time.Now()).
				Update(); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentRelapsed,
				IncidentID: incident.ID,
				ReportID:   &reportID,
				Note:       fmt.Sprintf("%s started overlapping again (run %s)", contractCode, reportUID),
				IsSystem:   true,
				At:         time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentRelapsed(*incident, incidentEvent.Note)
		}
	} else if incident != nil && incident.State == enums.IncidentStateOpen {
		if err := db.Model(Incident{}).
			Where(elm.Eq("id", incident.ID)).
			Set("state", enums.IncidentStateRecovered).
			Set("recovered_at", time.Now()).
			Update(); err != nil {
			return err
		}

		incidentEvent := IncidentEvent{
			Type:       enums.IncidentEventTypeIncidentRecovered,
			IncidentID: incident.ID,
			ReportID:   &reportID,
			Note:       fmt.Sprintf("%s stopped overlapping (run %s)", contractCode, reportUID),
			IsSystem:   true,
			At:         time.Now(),
		}
		if err := db.Save(&incidentEvent); err != nil {
			return err
		}

		go FireWebhookEventIncidentRecovered(*incident, incidentEvent.Note)
	}

	return nil
}

func ReviseIncidentForJobOvertime(contractID int64, contractCode string, reportID int64, reportUID string, ruleJobOvertime time.Duration, jobDuration time.Duration) error {
	db := database.GetDB()

	var incident *Incident
	if err := db.Model(Incident{}).
		Where(elm.And(
			elm.NotEq("state", enums.IncidentStateResolved),
			elm.Eq("trigger_type", enums.IncidentTriggerTypeJobOvertime),
			elm.Eq("contract_id", contractID),
		)).
		Scan(&incident); err != nil {
		return err
	}

	if jobDuration >= ruleJobOvertime {
		if incident == nil {
			incident = &Incident{
				State:       enums.IncidentStateOpen,
				TriggerType: enums.IncidentTriggerTypeJobOvertime,
				Title:       fmt.Sprintf("%s is overtime", contractCode),
				ContractID:  &contractID,
				OpenedAt:    time.Now(),
			}
			if err := db.Save(&incident); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentOpened,
				IncidentID: incident.ID,
				ReportID:   &reportID,
				Note:       fmt.Sprintf("%s went overtime, after %s (run %s)", contractCode, ruleJobOvertime, reportUID),
				IsSystem:   true,
				At:         time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentOpened(*incident, &incidentEvent.Note)
		} else if incident.State == enums.IncidentStateRecovered {
			if err := db.Model(Incident{}).
				Where(elm.Eq("id", incident.ID)).
				Set("state", enums.IncidentStateRelapsed).
				Set("relapsed_at", time.Now()).
				Update(); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentRelapsed,
				IncidentID: incident.ID,
				ReportID:   &reportID,
				Note:       fmt.Sprintf("%s went overtime again, after %s (run %s)", contractCode, ruleJobOvertime, reportUID),
				IsSystem:   true,
				At:         time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentRelapsed(*incident, incidentEvent.Note)
		}
	} else if incident != nil && incident.State == enums.IncidentStateOpen {
		if err := db.Model(Incident{}).
			Where(elm.Eq("id", incident.ID)).
			Set("state", enums.IncidentStateRecovered).
			Set("recovered_at", time.Now()).
			Update(); err != nil {
			return err
		}

		incidentEvent := IncidentEvent{
			Type:       enums.IncidentEventTypeIncidentRecovered,
			IncidentID: incident.ID,
			ReportID:   &reportID,
			Note:       fmt.Sprintf("%s finished in %s, overtime was %s (run %s)", contractCode, jobDuration, ruleJobOvertime, reportUID),
			IsSystem:   true,
			At:         time.Now(),
		}
		if err := db.Save(&incidentEvent); err != nil {
			return err
		}

		go FireWebhookEventIncidentRecovered(*incident, incidentEvent.Note)
	}

	return nil
}

func ReviseIncidentForStale(contractID int64, contractCode string, ruleStale time.Duration, isStale bool) error {
	db := database.GetDB()

	var incident *Incident
	if err := db.Model(Incident{}).
		Where(elm.And(
			elm.NotEq("state", enums.IncidentStateResolved),
			elm.Eq("trigger_type", enums.IncidentTriggerTypeStale),
			elm.Eq("contract_id", contractID),
		)).
		Scan(&incident); err != nil {
		return err
	}

	if isStale {
		if incident == nil {
			incident = &Incident{
				State:       enums.IncidentStateOpen,
				TriggerType: enums.IncidentTriggerTypeStale,
				Title:       fmt.Sprintf("%s is stale", contractCode),
				ContractID:  &contractID,
				OpenedAt:    time.Now(),
			}
			if err := db.Save(&incident); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentOpened,
				IncidentID: incident.ID,
				Note:       fmt.Sprintf("%s went stale, after %s", contractCode, ruleStale),
				IsSystem:   true,
				At:         time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentOpened(*incident, &incidentEvent.Note)
		} else if incident.State == enums.IncidentStateRecovered {
			if err := db.Model(Incident{}).
				Where(elm.Eq("id", incident.ID)).
				Set("state", enums.IncidentStateRelapsed).
				Set("relapsed_at", time.Now()).
				Update(); err != nil {
				return err
			}

			incidentEvent := IncidentEvent{
				Type:       enums.IncidentEventTypeIncidentRelapsed,
				IncidentID: incident.ID,
				Note:       fmt.Sprintf("%s went stale again, after %s", contractCode, ruleStale),
				IsSystem:   true,
				At:         time.Now(),
			}
			if err := db.Save(&incidentEvent); err != nil {
				return err
			}

			go FireWebhookEventIncidentRelapsed(*incident, incidentEvent.Note)
		}
	} else if incident != nil && incident.State == enums.IncidentStateOpen {
		if err := db.Model(Incident{}).
			Where(elm.Eq("id", incident.ID)).
			Set("state", enums.IncidentStateRecovered).
			Set("recovered_at", time.Now()).
			Update(); err != nil {
			return err
		}

		incidentEvent := IncidentEvent{
			Type:       enums.IncidentEventTypeIncidentRecovered,
			IncidentID: incident.ID,
			Note:       fmt.Sprintf("%s reported in time, within %s", contractCode, ruleStale),
			IsSystem:   true,
			At:         time.Now(),
		}
		if err := db.Save(&incidentEvent); err != nil {
			return err
		}

		go FireWebhookEventIncidentRecovered(*incident, incidentEvent.Note)
	}

	return nil
}

func ValidateContractRules(reportType enums.ContractType, rules map[enums.ContractRule]*time.Duration) error {
	allowedRules := AllowedContractRules(reportType)
	for rule := range rules {
		if !slices.Contains(allowedRules, rule) {
			return fmt.Errorf("rule %q is not allowed for %s", rule, reportType)
		}
	}

	check := func(order []enums.ContractRule) error {
		var prevRuleDuration time.Duration
		var prevRule enums.ContractRule
		for _, rule := range order {
			duration := rules[rule]
			if duration == nil {
				continue
			}
			if *duration != 0 {
				if prevRule != "" && prevRuleDuration > *duration {
					return log.Client.Errorf("%s (%s) must be less or equal to %s (%s)", prevRule, prevRuleDuration, rule, duration)
				}
				prevRuleDuration = *duration
				prevRule = rule
			}
		}
		return nil
	}

	if err := check([]enums.ContractRule{enums.ContractRuleStale, enums.ContractRuleStaleIncident}); err != nil {
		return err
	}

	if reportType == enums.ContractTypeJob {
		if err := check([]enums.ContractRule{enums.ContractRuleJobOvertimeIncident}); err != nil {
			return err
		}
	}

	return nil
}

func AllowedContractRules(reportType enums.ContractType) []enums.ContractRule {
	var allowedRules []enums.ContractRule

	allowedRules = append(allowedRules,
		enums.ContractRuleStale,
		enums.ContractRuleStaleIncident,
	)

	if reportType == enums.ContractTypeJob {
		allowedRules = append(allowedRules,
			enums.ContractRuleJobOvertimeIncident,
			enums.ContractRuleJobOverlapIncident,
		)
	}

	return allowedRules
}

func GenerateReportUID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[0:12]
}
