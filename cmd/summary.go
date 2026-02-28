package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
)

var SummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Print a one-line health summary.",
	RunE: func(cmd *cobra.Command, args []string) error {
		short, _ := cmd.Flags().GetBool("short")

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		summaryResponse, err := ipc.Send[dto.SummaryRequest, dto.SummaryResponse](conn, dto.SummaryRequest{})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		formatCodes := func(codes []string) string {
			if len(codes) <= 3 {
				return strings.Join(codes, ", ")
			}
			return strings.Join(codes[:3], ", ") + fmt.Sprintf(" +%d", len(codes)-3)
		}

		fmt.Printf("Timon — %d active incidents · %d critical%s · %d stale%s · %d warning · %d healthy · %d running jobs\n",
			summaryResponse.ActiveIncidents,
			len(summaryResponse.CriticalContracts),
			func() string {
				if short || len(summaryResponse.CriticalContracts) == 0 {
					return ""
				}
				return " (" + formatCodes(summaryResponse.CriticalContracts) + ")"
			}(),
			len(summaryResponse.StaleContracts),
			func() string {
				if short || len(summaryResponse.StaleContracts) == 0 {
					return ""
				}
				return " (" + formatCodes(summaryResponse.StaleContracts) + ")"
			}(),
			summaryResponse.NbWarningContracts,
			summaryResponse.NbHealthyContracts,
			summaryResponse.NbRunningJobs,
		)

		return nil
	},
}

func init() {
	SummaryCmd.Flags().Bool("short", false, "Short summary")
}
