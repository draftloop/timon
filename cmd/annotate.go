package cmd

import (
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/validations"
)

var AnnotateCmd = &cobra.Command{
	Use:   "annotate <INC-id> <note>",
	Short: "Annotate an incident.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		incidentId, err := validations.ParseIncidentCode(strings.TrimSpace(args[0]))
		if err != nil {
			return log.Client.Error(err.Error())
		}

		note := strings.TrimSpace(args[1])
		if err := validations.ValidateIncidentAnnotation(note); err != nil {
			return log.Client.Error(err.Error())
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.AnnotateRequest, dto.AnnotateResponse](conn, dto.AnnotateRequest{
			ID:   incidentId,
			Note: note,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}
