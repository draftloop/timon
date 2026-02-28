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
)

func statusIcon(s enums.Health) string {
	switch s {
	case enums.HealthHealthy:
		return "✓"
	case enums.HealthWarning:
		return "⚠"
	case enums.HealthCritical:
		return "✗"
	default:
		return "?"
	}
}

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active incidents and probes/jobs health.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		statusResponse, err := ipc.Send[dto.StatusRequest, dto.StatusResponse](conn, dto.StatusRequest{})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		fmt.Printf("ACTIVE INCIDENTS (%d)\n", len(statusResponse.ActiveIncidents))
		if len(statusResponse.ActiveIncidents) == 0 {
			fmt.Println("  None")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, inc := range statusResponse.ActiveIncidents {
				fmt.Fprintf(w, "  INC-%d\t%s\t%q\t%s ago\t%s\n",
					inc.ID,
					inc.State,
					inc.Title,
					utils.HumanDuration(time.Since(inc.OpenedAt)),
					func() string {
						if inc.IsManual {
							return "(manual)"
						}
						return ""
					}(),
				)
			}
			w.Flush()
		}

		fmt.Println()

		fmt.Printf("PROBES & JOBS (%d)\n", len(statusResponse.Contracts))
		if len(statusResponse.Contracts) == 0 {
			fmt.Println("  None")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, contract := range statusResponse.Contracts {
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
					contract.Code,
					func() string {
						v := ""
						if contract.LastReport != nil {
							if contract.Type == enums.ContractTypeJob && contract.LastReport.JobEndedAt == nil {
								v += "● running " + contract.LastReport.UID
								if contract.LastReport.Health != enums.HealthUnknown {
									v += " (" + statusIcon(contract.LastReport.Health) + " " + string(contract.LastReport.Health) + ")"
								}
							} else {
								v += statusIcon(contract.LastReport.Health) + " " + string(contract.LastReport.Health)
							}
							if contract.IsStale {
								v = "stale (" + v + ")"
							}

							if contract.Type == enums.ContractTypeJob {
								v += " — " + utils.HumanDuration(time.Since(*contract.LastReport.JobStartedAt)) + " ago"
							} else {
								v += " — " + utils.HumanDuration(time.Since(*contract.LastReport.ProbePushedAt)) + " ago"
							}
						} else {
							v += statusIcon(enums.HealthUnknown) + " " + string(enums.HealthUnknown)
						}
						return v
					}(),
					func() string {
						v := ""
						if contract.LastReport != nil {
							if contract.LastReport.ProbeComment != nil {
								v = fmt.Sprintf("%q", *contract.LastReport.ProbeComment)
							} else if contract.LastReport.JobEndComment != nil {
								v = fmt.Sprintf("%q", *contract.LastReport.JobEndComment)
							} else if contract.LastReport.JobStartComment != nil {
								v = fmt.Sprintf("%q", *contract.LastReport.JobStartComment)
							}
						}
						return v
					}(),
					func() string {
						var v []string
						for _, id := range contract.IncidentsID {
							v = append(v, fmt.Sprintf("INC-%d", id))
						}
						return strings.Join(v, ", ")
					}(),
				)
			}
			w.Flush()
		}

		return nil
	},
}
