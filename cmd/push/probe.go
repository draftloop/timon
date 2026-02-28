package cmdpush

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	"timon/internal/enums"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/utils"
	"timon/internal/validations"
)

var PushProbeCmd = &cobra.Command{
	Use:   "probe <code> <healthy|warning|critical>",
	Short: "Push a probe.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := strings.TrimSpace(args[0])
		if _, _, err := validations.ParseContractCode(code, false); err != nil {
			return log.Client.Error(err.Error())
		}

		healthStr := strings.TrimSpace(args[1])
		health, err := enums.ParseHealth(healthStr)
		if err != nil {
			return log.Client.Error(err.Error())
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

		rules := dto.PushProbeRequestRules{}

		if s, _ := cmd.Flags().GetString("stale-after"); strings.TrimSpace(s) != "" {
			d, err := utils.ParseDuration(strings.TrimSpace(s))
			if err != nil {
				return fmt.Errorf("invalid rule stale-after — %s", err)
			}
			rules.Stale = &d
		}

		if s, _ := cmd.Flags().GetString("stale-incident-after"); strings.TrimSpace(s) != "" {
			d, err := utils.ParseDuration(strings.TrimSpace(s))
			if err != nil {
				return fmt.Errorf("invalid rule stale-incident-after — %s", err)
			}
			rules.StaleIncident = &d
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.PushProbeRequest, dto.PushProbeResponse](conn, dto.PushProbeRequest{
			Code:    code,
			Health:  health,
			Comment: comment,
			Rules:   rules,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}

func init() {
	PushProbeCmd.Flags().String("comment", "", "Add an comment")
	PushProbeCmd.Flags().String("stale-after", "", "Delay before the probe is flagged as stale if the next push does not occur")
	PushProbeCmd.Flags().String("stale-incident-after", "", "Same as --stale-after, and also opens an incident")
}
