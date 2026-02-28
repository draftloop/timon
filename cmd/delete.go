package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/validations"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete <probe-code|job-code|probe-code:sample-uid|job-code:run-uid|INC-id>",
	Short: "Delete a probe, a job, a sample, a run, or an incident.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := strings.TrimSpace(args[0])
		if _, err := validations.ParseIncidentCode(code); err == nil {
			// ok
		} else if _, _, err := validations.ParseContractCode(code, true); err != nil {
			return log.Client.Error(err.Error())
		}

		force, _ := cmd.Flags().GetBool("force")

		cmd.SilenceUsage = true

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			fmt.Printf("Are you sure you want to delete %q? [y/N] ", code)

			var confirm string
			fmt.Scanln(&confirm)

			if strings.ToLower(confirm) != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.DeleteRequest, dto.DeleteResponse](conn, dto.DeleteRequest{
			Code:  code,
			Force: force,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}

func init() {
	DeleteCmd.Flags().Bool("force", false, "Bypass active incident protection")
	DeleteCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
}
