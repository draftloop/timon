package cmdpush

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/validations"
)

var PushIncidentCmd = &cobra.Command{
	Use:   "incident <title> [description]",
	Short: "Push a new incident. Returns an incident code.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.TrimSpace(args[0])
		if err := validations.ValidateIncidentTitle(title); err != nil {
			return log.Client.Error(err.Error())
		}

		var description *string
		if len(args) == 2 {
			v := strings.TrimSpace(args[1])
			if v != "" {
				description = &v
				if err := validations.ValidateIncidentDescription(*description); err != nil {
					return log.Client.Error(err.Error())
				}
			}
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		pushIncidentResponse, err := ipc.Send[dto.PushIncidentRequest, dto.PushIncidentResponse](conn, dto.PushIncidentRequest{
			Title:       title,
			Description: description,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		fmt.Printf("INC-%d", pushIncidentResponse.ID)

		return nil
	},
}
