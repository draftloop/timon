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

		parts := []string{
			fmt.Sprintf("%d active incidents", motdResponse.ActiveIncidents),
		}

		if n := len(motdResponse.CriticalContracts); n > 0 {
			parts = append(parts, fmt.Sprintf("%d critical (%s)", n, formatCodes(motdResponse.CriticalContracts)))
		}

		if n := len(motdResponse.StaleContracts); n > 0 {
			parts = append(parts, fmt.Sprintf("%d stale (%s)", n, formatCodes(motdResponse.StaleContracts)))
		}

		if n := motdResponse.NbWarningContracts; n > 0 {
			parts = append(parts, fmt.Sprintf("%d warning", n))
		}

		if n := motdResponse.NbHealthyContracts; n > 0 {
			parts = append(parts, fmt.Sprintf("%d healthy", n))
		}

		parts = append(parts, fmt.Sprintf("%d running jobs", motdResponse.NbRunningJobs))

		fmt.Printf("Timon — %s\n", strings.Join(parts, " · "))

		return nil
	},
}
