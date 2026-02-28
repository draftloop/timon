package cmdpushjob

import (
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/validations"
)

var PushJobEndCmd = &cobra.Command{
	Use:   "end <code:run-uid>",
	Short: "End a job run.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var contractCode string
		var reportUID string
		var err error
		if contractCode, reportUID, err = validations.ParseContractCode(strings.TrimSpace(args[0]), true); err != nil {
			return log.Client.Error(err.Error())
		} else if reportUID == "" {
			return log.Client.Errorf("invalid run UID: %s", args[0])
		}

		var comment *string
		commentStr, _ := cmd.Flags().GetString("comment")
		commentStr = strings.TrimSpace(commentStr)
		if commentStr != "" {
			if err := validations.ValidateReportComment(commentStr); err != nil {
				return log.Client.Error(err.Error())
			}
			comment = &commentStr
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.PushJobEndRequest, dto.PushJobEndResponse](conn, dto.PushJobEndRequest{
			Code:    contractCode,
			RunUID:  reportUID,
			Comment: comment,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}

func init() {
	PushJobEndCmd.Flags().String("comment", "", "Add an end comment")
}
