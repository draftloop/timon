package handlers

import (
	"fmt"
	"github.com/draftloop/elm"
	"strconv"
	"strings"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
	"timon/internal/enums"
	"timon/internal/ipc/daemon/handler"
	"timon/internal/ipc/dto"
	"timon/internal/validations"
)

func ShowHandler(req dto.ShowRequest) (res handler.Response[dto.ShowResponse]) {
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
			Where(elm.Eq("Incident.id", incidentId)).
			Scan(&incident); err != nil {
			return res.SendDaemonError(err)
		} else if incident == nil {
			return res.SendClientError(fmt.Errorf("incident does not exist"))
		}

		var events []models.IncidentEvent
		if err := db.Model(models.IncidentEvent{}).
			Where(elm.Eq("incident_id", incidentId)).
			Scan(&events); err != nil {
			return res.SendDaemonError(err)
		}

		return res.Send(dto.ShowResponse{
			Incident: &dto.ShowResponseIncident{
				ID:          incident.ID,
				State:       incident.State,
				TriggerType: incident.TriggerType,
				Title:       incident.Title,
				Description: incident.Description,
				ContractID:  incident.ContractID,
				OpenedAt:    incident.OpenedAt,
				RecoveredAt: incident.RecoveredAt,
				RelapsedAt:  incident.RelapsedAt,
				ResolvedAt:  incident.ResolvedAt,
				Timeline: func() []dto.ShowResponseIncidentEvent {
					list := make([]dto.ShowResponseIncidentEvent, 0, len(events))
					for _, event := range events {
						list = append(list, dto.ShowResponseIncidentEvent{
							Type:     event.Type,
							ReportID: event.ReportID,
							Note:     event.Note,
							IsSystem: event.IsSystem,
							At:       event.At,
						})
					}
					return list
				}(),
			},
		})
	} else if reportUID != "" {
		type SelectReport struct {
			Report    *models.Report
			Contract  *models.Contract
			Incidents string
		}
		var row SelectReport
		if err := db.Model(models.Report{}).
			SelectAllFrom(models.Report{}).
			SelectAllFrom(models.Contract{}).
			UnsafeSelect("(SELECT GROUP_CONCAT(Incident.id) FROM incident_events IncidentEvent INNER JOIN incidents Incident ON Incident.id = IncidentEvent.incident_id WHERE IncidentEvent.report_id = Report.id AND Incident.state != \"" + string(enums.IncidentStateResolved) + "\") as Incidents").
			InnerRelation(models.Contract{}).
			Where(elm.And(
				elm.Eq("Contract.code", contractCode),
				elm.Eq("Report.uid", reportUID),
			)).
			Scan(&row); err != nil {
			return res.SendDaemonError(err)
		} else if row.Report == nil {
			return res.SendClientError(fmt.Errorf("sample or run does not exist"))
		}

		var steps []models.ReportJobStep
		if row.Contract.Type == enums.ContractTypeJob {
			if err := db.Model(models.ReportJobStep{}).
				Where(elm.Eq("report_id", row.Report.ID)).
				UnsafeOrderBy("id ASC").
				Scan(&steps); err != nil {
				return res.SendDaemonError(err)
			}
		}

		return res.Send(dto.ShowResponse{
			Report: &dto.ShowResponseReport{
				Code: row.Contract.Code,
				Type: row.Contract.Type,
				ShowResponseContractReport: dto.ShowResponseContractReport{
					UID:                     row.Report.UID,
					Health:                  row.Report.Health,
					ProbePushedAt:           row.Report.ProbePushedAt,
					ProbeComment:            row.Report.ProbeComment,
					JobStartedAt:            row.Report.JobStartedAt,
					JobStartComment:         row.Report.JobStartComment,
					JobEndedAt:              row.Report.JobEndedAt,
					JobEndComment:           row.Report.JobEndComment,
					RuleStale:               row.Report.RuleStale,
					RuleStaleIncident:       row.Report.RuleStaleIncident,
					RuleJobOvertimeIncident: row.Report.RuleJobOvertimeIncident,
					RuleJobOverlapIncident:  row.Report.RuleJobOverlapIncident,
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
				},
				Steps: func() []dto.ShowResponseReportJobRunStep {
					list := make([]dto.ShowResponseReportJobRunStep, 0, len(steps))
					for _, step := range steps {
						list = append(list, dto.ShowResponseReportJobRunStep{
							Label:  step.Label,
							Health: step.Health,
							At:     step.At,
						})
					}
					return list
				}(),
			},
		})
	} else {
		type SelectContract struct {
			Contract   *models.Contract
			LastReport *models.Report
			Incidents  string
		}
		var row SelectContract
		if err := db.Model(models.Contract{}).
			SelectAllFrom(models.Contract{}).
			SelectAllFromAs(models.Report{}, "LastReport").
			UnsafeSelect("(SELECT GROUP_CONCAT(Incident.id) FROM incidents Incident WHERE Incident.contract_id = Contract.id AND Incident.state != \"" + string(enums.IncidentStateResolved) + "\") as Incidents").
			LeftRelation(models.Report{}).
			Where(elm.Eq("code", contractCode)).
			Scan(&row); err != nil {
			return res.SendDaemonError(err)
		} else if row.Contract == nil {
			return res.SendClientError(fmt.Errorf("probe or job does not exist"))
		}

		type SelectReports struct {
			Report    models.Report
			Incidents string
		}
		var reports []SelectReports
		if row.LastReport != nil {
			if err := db.Model(models.Report{}).
				SelectAllFrom(models.Report{}).
				UnsafeSelect("(SELECT GROUP_CONCAT(Incident.id) FROM incident_events IncidentEvent INNER JOIN incidents Incident ON Incident.id = IncidentEvent.incident_id WHERE IncidentEvent.report_id = Report.id AND Incident.state != \"" + string(enums.IncidentStateResolved) + "\") as Incidents").
				Where(elm.Eq("contract_id", row.Contract.ID)).
				UnsafeOrderBy("id DESC").
				Scan(&reports); err != nil {
				return res.SendDaemonError(err)
			}
		}

		return res.Send(dto.ShowResponse{
			Contract: &dto.ShowResponseContract{
				Code:    row.Contract.Code,
				Type:    row.Contract.Type,
				IsStale: row.Contract.IsStale,
				LastReport: func() *dto.ShowResponseContractReport {
					if row.LastReport == nil {
						return nil
					}
					return &dto.ShowResponseContractReport{
						UID:                     row.LastReport.UID,
						Health:                  row.LastReport.Health,
						ProbePushedAt:           row.LastReport.ProbePushedAt,
						ProbeComment:            row.LastReport.ProbeComment,
						JobStartedAt:            row.LastReport.JobStartedAt,
						JobStartComment:         row.LastReport.JobStartComment,
						JobEndedAt:              row.LastReport.JobEndedAt,
						JobEndComment:           row.LastReport.JobEndComment,
						RuleStale:               row.LastReport.RuleStale,
						RuleStaleIncident:       row.LastReport.RuleStaleIncident,
						RuleJobOvertimeIncident: row.LastReport.RuleJobOvertimeIncident,
						RuleJobOverlapIncident:  row.LastReport.RuleJobOverlapIncident,
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
					}
				}(),
				Reports: func() []dto.ShowResponseContractReport {
					list := make([]dto.ShowResponseContractReport, 0, len(reports))
					for _, report := range reports {
						list = append(list, dto.ShowResponseContractReport{
							UID:                     report.Report.UID,
							Health:                  report.Report.Health,
							ProbePushedAt:           report.Report.ProbePushedAt,
							ProbeComment:            report.Report.ProbeComment,
							JobStartedAt:            report.Report.JobStartedAt,
							JobStartComment:         report.Report.JobStartComment,
							JobEndedAt:              report.Report.JobEndedAt,
							JobEndComment:           report.Report.JobEndComment,
							RuleStale:               report.Report.RuleStale,
							RuleStaleIncident:       report.Report.RuleStaleIncident,
							RuleJobOvertimeIncident: report.Report.RuleJobOvertimeIncident,
							RuleJobOverlapIncident:  report.Report.RuleJobOverlapIncident,
							IncidentsID: func() []int64 {
								if report.Incidents == "" {
									return nil
								}
								split := strings.Split(report.Incidents, ",")
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
			},
		})
	}
}
