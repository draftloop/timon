package cmd

import (
	"fmt"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"

	"github.com/spf13/cobra"
)

var MotdCmd = &cobra.Command{
	Use:   "motd",
	Short: "Print a concise health overview.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		motdResponse, err := ipc.Send[dto.MotdRequest, dto.MotdResponse](conn, dto.MotdRequest{})
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
			motdResponse.ActiveIncidents,
			len(motdResponse.CriticalContracts),
			func() string {
				if len(motdResponse.CriticalContracts) == 0 {
					return ""
				}
				return " (" + formatCodes(motdResponse.CriticalContracts) + ")"
			}(),
			len(motdResponse.StaleContracts),
			func() string {
				if len(motdResponse.StaleContracts) == 0 {
					return ""
				}
				return " (" + formatCodes(motdResponse.StaleContracts) + ")"
			}(),
			motdResponse.NbWarningContracts,
			motdResponse.NbHealthyContracts,
			motdResponse.NbRunningJobs,
		)

		return nil
	},
}
