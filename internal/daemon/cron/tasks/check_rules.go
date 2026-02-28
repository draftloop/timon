package tasks

import (
	"fmt"
	"github.com/draftloop/elm"
	"time"
	"timon/internal/daemon/cron/ctx"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
)

func CheckRules(taskContext ctx.TaskContext) error {
	db := database.GetDB()

	var staleRows []models.Contract
	if err := db.Model(models.Contract{}).
		InnerRelation(models.Report{}).
		Where(elm.Or(
			elm.LtEq("rule_stale_incident_at", time.Now()),
			elm.LtEq("rule_stale_at", time.Now()),
		)).
		UnsafeOrderBy("rule_stale_incident_at asc, rule_stale_at asc").
		Scan(&staleRows); err != nil {
		return fmt.Errorf("error fetching reports: %w", err)
	}
	for _, row := range staleRows {
		if row.RuleStaleIncidentAt != nil && row.RuleStaleIncidentAt.Before(time.Now()) {
			if err := db.Model(models.Contract{}).
				Set("rule_stale_at", nil).
				Set("is_stale", true).
				Set("rule_stale_incident_at", nil).
				Where(elm.Eq("id", row.ID)).
				Update(); err != nil {
				return fmt.Errorf("error updating contract: %w", err)
			}
			if err := models.ReviseIncidentForStale(row.ID, row.Code, *row.RuleStaleIncident, true); err != nil {
				return fmt.Errorf("error revise incident for stale: %w", err)
			}
		} else if row.RuleStaleAt != nil && row.RuleStaleAt.Before(time.Now()) {
			if err := db.Model(models.Contract{}).
				Set("rule_stale_at", nil).
				Set("is_stale", true).
				Where(elm.Eq("id", row.ID)).
				Update(); err != nil {
				return fmt.Errorf("error updating contract: %w", err)
			}
		}
	}

	var overtimeRows []models.Report
	if err := db.Model(models.Report{}).
		InnerRelation(models.Contract{}).
		Where(elm.And(
			elm.Eq("Contract.type", enums.ContractTypeJob),
			elm.IsNull("job_ended_at"),
			elm.Or(
				elm.LtEq("rule_job_overtime_incident_at", time.Now()),
			),
		)).
		UnsafeOrderBy("rule_job_overtime_incident_at asc").
		Scan(&overtimeRows); err != nil {
		return fmt.Errorf("error fetching job reports: %w", err)
	}
	for _, row := range overtimeRows {
		if row.RuleJobOvertimeIncidentAt != nil && row.RuleJobOvertimeIncidentAt.Before(time.Now()) {
			if err := db.Model(models.Report{}).
				Set("rule_job_overtime_incident_at", nil).
				Where(elm.Eq("id", row.ID)).
				Update(); err != nil {
				return fmt.Errorf("error updating report: %w", err)
			}
			if err := models.ReviseIncidentForJobOvertime(row.Contract.ID, row.Contract.Code, row.ID, row.UID, *row.RuleJobOvertimeIncident, *row.RuleJobOvertimeIncident); err != nil {
				return fmt.Errorf("error revise incident for job overtime: %w", err)
			}
		}
	}

	return nil
}
