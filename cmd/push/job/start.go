package cmdpushjob

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/utils"
	"timon/internal/validations"
)

var PushJobStartCmd = &cobra.Command{
	Use:   "start <code>",
	Short: "Push a new job run. Returns a run uid.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := strings.TrimSpace(args[0])
		if _, _, err := validations.ParseContractCode(code, false); err != nil {
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

		rules := dto.PushJobStartRequestRules{}

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

		if s, _ := cmd.Flags().GetString("overtime-incident-after"); strings.TrimSpace(s) != "" {
			d, err := utils.ParseDuration(strings.TrimSpace(s))
			if err != nil {
				return fmt.Errorf("invalid rule overtime-incident-after — %s", err)
			}
			rules.JobOvertimeIncident = &d
		}

		ruleJobOverlapIncident, _ := cmd.Flags().GetBool("overlap-incident")
		rules.JobOverlapIncident = &ruleJobOverlapIncident

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		pushJobStartResponse, err := ipc.Send[dto.PushJobStartRequest, dto.PushJobStartResponse](conn, dto.PushJobStartRequest{
			Code:    code,
			Comment: comment,
			Rules:   rules,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		fmt.Print(pushJobStartResponse.RunUID)

		return nil
	},
}

func init() {
	PushJobStartCmd.Flags().String("comment", "", "Add a start comment")
	PushJobStartCmd.Flags().String("stale-after", "", "Delay before the job is flagged as stale if the next push does not occur")
	PushJobStartCmd.Flags().String("stale-incident-after", "", "Same as --stale-after, and also opens an incident")
	PushJobStartCmd.Flags().String("overtime-incident-after", "", "Delay before a new incident is created if the job is overtime")
	PushJobStartCmd.Flags().Bool("overlap-incident", true, "Create an incident when a job overlap is detected")
}
