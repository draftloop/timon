package cmd

import (
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/validations"
)

var ResolveCmd = &cobra.Command{
	Use:   "resolve <INC-id>",
	Short: "Resolve an incident.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		incidentId, err := validations.ParseIncidentCode(strings.TrimSpace(args[0]))
		if err != nil {
			return log.Client.Error(err.Error())
		}

		var note *string
		noteStr, _ := cmd.Flags().GetString("note")
		noteStr = strings.TrimSpace(noteStr)
		if noteStr != "" {
			if err := validations.ValidateIncidentAnnotation(noteStr); err != nil {
				return log.Client.Error(err.Error())
			}
			note = &noteStr
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.ResolveRequest, dto.ResolveResponse](conn, dto.ResolveRequest{
			ID:   incidentId,
			Note: note,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}

func init() {
	ResolveCmd.Flags().String("note", "", "Annotate the incident before resolving it")
}
