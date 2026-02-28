package cmdpushjob

import (
	"github.com/spf13/cobra"
	"strings"
	"timon/internal/enums"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/validations"
)

var PushJobSepCmd = &cobra.Command{
	Use:   "step <code:run-uid> <label> <healthy|warning|critical>",
	Short: "Push a step to a job run.",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		var contractCode string
		var reportUID string
		var err error
		if contractCode, reportUID, err = validations.ParseContractCode(strings.TrimSpace(args[0]), true); err != nil {
			return log.Client.Error(err.Error())
		} else if reportUID == "" {
			return log.Client.Errorf("invalid run UID: %s", args[0])
		}

		label := strings.TrimSpace(args[1])
		if err := validations.ValidateReportJobLabel(label); err != nil {
			return log.Client.Error(err.Error())
		}

		healthStr := strings.TrimSpace(args[2])
		health, err := enums.ParseHealth(healthStr)
		if err != nil {
			return log.Client.Error(err.Error())
		}

		end, _ := cmd.Flags().GetBool("end")

		var endComment *string
		if end {
			endCommentStr, _ := cmd.Flags().GetString("end-comment")
			endCommentStr = strings.TrimSpace(endCommentStr)
			if endCommentStr != "" {
				if err := validations.ValidateReportComment(endCommentStr); err != nil {
					return log.Client.Error(err.Error())
				}
				endComment = &endCommentStr
			}
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.PushJobStepRequest, dto.PushJobStepResponse](conn, dto.PushJobStepRequest{
			Code:       contractCode,
			Label:      label,
			Health:     health,
			RunUID:     reportUID,
			End:        end,
			EndComment: endComment,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}

func init() {
	PushJobSepCmd.Flags().Bool("end", false, "End run after step")
	PushJobSepCmd.Flags().String("end-comment", "", "Add an end comment")
}
