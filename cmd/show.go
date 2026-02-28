package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"timon/internal/enums"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/utils"
	"timon/internal/validations"
)

var ShowCmd = &cobra.Command{
	Use:   "show <probe-code:sample-uid|job-code:run-uid|INC-id>",
	Short: "Show details of a probe, probe sample, job, job run, or incident.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := strings.TrimSpace(args[0])
		if _, err := validations.ParseIncidentCode(code); err == nil {
			// ok
		} else if _, _, err := validations.ParseContractCode(code, true); err != nil {
			return log.Client.Error(err.Error())
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		showResponse, err := ipc.Send[dto.ShowRequest, dto.ShowResponse](conn, dto.ShowRequest{
			Code: code,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		if showResponse.Incident != nil {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "INCIDENT  INC-%d\n", showResponse.Incident.ID)
			fmt.Fprintf(w, "  state\t%s\n", showResponse.Incident.State)
			if showResponse.Incident.TriggerType == enums.IncidentTriggerTypeManual {
				fmt.Fprintf(w, "  trigger type\tmanual\n")
			}
			fmt.Fprintf(w, "  title\t%s\n", showResponse.Incident.Title)
			if showResponse.Incident.Description != nil {
				fmt.Fprintf(w, "  description\t%s\n", *showResponse.Incident.Description)
			}
			fmt.Fprintf(w, "  opened\t%s ago\n", utils.HumanDuration(time.Since(showResponse.Incident.OpenedAt)))
			if showResponse.Incident.RecoveredAt != nil {
				fmt.Fprintf(w, "  recovered\t%s ago\n", utils.HumanDuration(time.Since(*showResponse.Incident.RecoveredAt)))
			}
			if showResponse.Incident.RelapsedAt != nil {
				fmt.Fprintf(w, "  relapsed\t%s ago\n", utils.HumanDuration(time.Since(*showResponse.Incident.RelapsedAt)))
			}
			if showResponse.Incident.ResolvedAt != nil {
				fmt.Fprintf(w, "  resolved\t%s ago\n", utils.HumanDuration(time.Since(*showResponse.Incident.ResolvedAt)))
			}
			w.Flush()

			fmt.Println()

			w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TIMELINE\n")
			if len(showResponse.Incident.Timeline) == 0 {
				fmt.Fprintf(w, "  None\n")
			}
			for _, event := range showResponse.Incident.Timeline {
				fmt.Fprintf(w, "  +%s\t%s\t%s\t%q\n",
					utils.HumanDuration(event.At.Sub(showResponse.Incident.OpenedAt)),
					strings.ReplaceAll(string(event.Type), "incident_", ""),
					func() string {
						if event.IsSystem {
							return "[system]"
						}
						return "[human]"
					}(),
					event.Note,
				)
			}
			w.Flush()
		} else if showResponse.Contract != nil {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "%s  %s\n", strings.ToUpper(string(showResponse.Contract.Type)), showResponse.Contract.Code)
			fmt.Fprintf(w, "  health\t%s\n", func() string {
				if showResponse.Contract.LastReport == nil {
					return statusIcon(enums.HealthUnknown) + " " + string(enums.HealthUnknown)
				}
				v := ""
				if showResponse.Contract.Type == enums.ContractTypeJob && showResponse.Contract.LastReport.JobEndedAt == nil {
					v += "● running " + showResponse.Contract.LastReport.UID
					if showResponse.Contract.LastReport.Health != enums.HealthUnknown {
						v += " (" + statusIcon(showResponse.Contract.LastReport.Health) + " " + string(showResponse.Contract.LastReport.Health) + ")"
					}
				} else {
					v += statusIcon(showResponse.Contract.LastReport.Health) + " " + string(showResponse.Contract.LastReport.Health)
				}
				if showResponse.Contract.IsStale {
					v = "stale (" + v + ")"
				}
				return v
			}())
			if showResponse.Contract.LastReport != nil {
				if showResponse.Contract.Type == enums.ContractTypeJob {
					if showResponse.Contract.LastReport.JobStartedAt != nil {
						fmt.Fprintf(w, "  last start\t%s\n", utils.HumanDuration(time.Since(*showResponse.Contract.LastReport.JobStartedAt))+" ago")
					}
				} else {
					if showResponse.Contract.LastReport.ProbePushedAt != nil {
						fmt.Fprintf(w, "  last push\t%s\n", utils.HumanDuration(time.Since(*showResponse.Contract.LastReport.ProbePushedAt))+" ago")
					}
				}
				if showResponse.Contract.LastReport.RuleStale != nil {
					fmt.Fprintf(w, "  stale after\t%s\n", utils.HumanDuration(*showResponse.Contract.LastReport.RuleStale))
				}
				if showResponse.Contract.LastReport.RuleStaleIncident != nil {
					fmt.Fprintf(w, "  stale incident after\t%s\n", utils.HumanDuration(*showResponse.Contract.LastReport.RuleStaleIncident))
				}
				if showResponse.Contract.LastReport.RuleJobOvertimeIncident != nil {
					fmt.Fprintf(w, "  overtime incident after\t%s\n", utils.HumanDuration(*showResponse.Contract.LastReport.RuleJobOvertimeIncident))
				}
				if showResponse.Contract.LastReport.RuleJobOverlapIncident != nil && *showResponse.Contract.LastReport.RuleJobOverlapIncident {
					fmt.Fprintf(w, "  overlap incident\t%s\n", "yes")
				}
				if len(showResponse.Contract.LastReport.IncidentsID) > 0 {
					fmt.Fprintf(w, "  active incidents\t%s\n",
						func() string {
							var v []string
							for _, id := range showResponse.Contract.LastReport.IncidentsID {
								v = append(v, fmt.Sprintf("INC-%d", id))
							}
							return strings.Join(v, ", ")
						}(),
					)
				}

			}
			w.Flush()

			fmt.Println()

			w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if showResponse.Contract.Type == enums.ContractTypeJob {
				fmt.Fprintf(w, "RUNS\n")
				if len(showResponse.Contract.Reports) == 0 {
					fmt.Fprintf(w, "  None\n")
				}
				for _, report := range showResponse.Contract.Reports {
					fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n",
						report.UID,
						func() string {
							v := ""
							if report.JobEndedAt == nil {
								v += "● running"
								if report.Health != enums.HealthUnknown {
									v += " (" + statusIcon(report.Health) + " " + string(report.Health) + ")"
								}
							} else {
								v += statusIcon(report.Health) + " " + string(report.Health)
							}
							return v
						}(),
						func() string {
							if report.JobEndedAt != nil {
								return "ended " + utils.HumanDuration(time.Since(*report.JobEndedAt)) + " ago (" + utils.HumanDuration(report.JobEndedAt.Sub(*report.JobStartedAt)) + ")"
							} else if report.JobStartedAt != nil {
								return "started " + utils.HumanDuration(time.Since(*report.JobStartedAt)) + " ago"
							}
							return ""
						}(),
						func() string {
							v := ""
							if report.JobEndComment != nil {
								v = *report.JobEndComment
							} else if report.JobStartComment != nil {
								v = *report.JobStartComment
							} else {
								return ""
							}
							return fmt.Sprintf("%q", v)
						}(),
						func() string {
							var v []string
							for _, id := range report.IncidentsID {
								v = append(v, fmt.Sprintf("INC-%d", id))
							}
							return strings.Join(v, ", ")
						}(),
					)
				}
			} else {
				fmt.Fprintf(w, "SAMPLES\n")
				if len(showResponse.Contract.Reports) == 0 {
					fmt.Fprintf(w, "  None\n")
				}
				for _, report := range showResponse.Contract.Reports {
					fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
						report.UID,
						statusIcon(report.Health)+" "+string(report.Health),
						func() string {
							if report.ProbeComment != nil {
								return fmt.Sprintf("%q", *report.ProbeComment)
							}
							return ""
						}(),
						func() string {
							if report.ProbePushedAt != nil {
								return utils.HumanDuration(time.Since(*report.ProbePushedAt)) + " ago"
							}
							return ""
						}(),
					)
				}
			}
			w.Flush()
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "%s  %s  %s %s\n", strings.ToUpper(string(showResponse.Report.Type)), showResponse.Report.Code, func() string {
				if showResponse.Report.Type == enums.ContractTypeJob {
					return "run"
				}
				return "sample"
			}(), showResponse.Report.UID)
			fmt.Fprintf(w, "  health\t%s\n", func() string {
				v := ""
				if showResponse.Report.Type == enums.ContractTypeJob && showResponse.Report.JobEndedAt == nil {
					v += "● running"
					if showResponse.Report.Health != enums.HealthUnknown {
						v += " (" + statusIcon(showResponse.Report.Health) + " " + string(showResponse.Report.Health) + ")"
					}
				} else {
					v += statusIcon(showResponse.Report.Health) + " " + string(showResponse.Report.Health)
				}
				return v
			}())
			if showResponse.Report.Type == enums.ContractTypeJob {
				fmt.Fprintf(w, "  started\t%s\n", utils.HumanDuration(time.Since(*showResponse.Report.JobStartedAt))+" ago")
				if showResponse.Report.JobStartComment != nil {
					fmt.Fprintf(w, "  start comment\t%s\n", *showResponse.Report.JobStartComment)
				}
				if showResponse.Report.JobEndedAt != nil {
					fmt.Fprintf(w, "  duration\t%s\n", utils.HumanDuration(showResponse.Report.JobEndedAt.Sub(*showResponse.Report.JobStartedAt)))
				}
				if showResponse.Report.JobEndComment != nil {
					fmt.Fprintf(w, "  end comment\t%s\n", *showResponse.Report.JobEndComment)
				}
			} else {
				fmt.Fprintf(w, "  pushed\t%s\n", utils.HumanDuration(time.Since(*showResponse.Report.ProbePushedAt))+" ago")
			}
			if showResponse.Report.RuleStale != nil {
				fmt.Fprintf(w, "  stale after\t%s\n", utils.HumanDuration(*showResponse.Report.RuleStale))
			}
			if showResponse.Report.RuleStaleIncident != nil {
				fmt.Fprintf(w, "  stale incident after\t%s\n", utils.HumanDuration(*showResponse.Report.RuleStaleIncident))
			}
			if showResponse.Report.RuleJobOvertimeIncident != nil {
				fmt.Fprintf(w, "  overtime incident after\t%s\n", utils.HumanDuration(*showResponse.Report.RuleJobOvertimeIncident))
			}
			if showResponse.Report.RuleJobOverlapIncident != nil && *showResponse.Report.RuleJobOverlapIncident {
				fmt.Fprintf(w, "  overlap incident\t%s\n", "yes")
			}
			if len(showResponse.Report.IncidentsID) > 0 {
				fmt.Fprintf(w, "  active incidents\t%s\n",
					func() string {
						var v []string
						for _, id := range showResponse.Report.IncidentsID {
							v = append(v, fmt.Sprintf("INC-%d", id))
						}
						return strings.Join(v, ", ")
					}(),
				)
			}
			w.Flush()

			if showResponse.Report.Type == enums.ContractTypeJob {
				fmt.Println()

				w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintf(w, "STEPS\n")
				if len(showResponse.Report.Steps) == 0 {
					fmt.Fprintf(w, "  None\n")
				}
				for _, step := range showResponse.Report.Steps {
					fmt.Fprintf(w, "  +%s\t%s\t%q\n", utils.HumanDuration(step.At.Sub(*showResponse.Report.JobStartedAt)), statusIcon(step.Health)+" "+string(step.Health), step.Label)
				}
				w.Flush()
			}
		}

		return nil
	},
}
