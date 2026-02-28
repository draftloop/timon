package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	"time"
	ipc "timon/internal/ipc/client"
	"timon/internal/ipc/dto"
	"timon/internal/log"
	"timon/internal/utils"
	"timon/internal/validations"
)

var TruncateCmd = &cobra.Command{
	Use:   "truncate [<probe-code|job-code>]",
	Short: "Truncate old samples, runs, and resolved incidents based on retention durations.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var code *string
		if len(args) == 1 {
			code = utils.Ptr(strings.TrimSpace(args[0]))
			if _, _, err := validations.ParseContractCode(*code, false); err != nil {
				return log.Client.Error(err.Error())
			}
		}

		keepStr, _ := cmd.Flags().GetString("keep")
		keepStr = strings.TrimSpace(keepStr)

		keepHealthyStr, _ := cmd.Flags().GetString("keep-healthy")
		keepHealthyStr = strings.TrimSpace(keepHealthyStr)

		keepWarningStr, _ := cmd.Flags().GetString("keep-warning")
		keepWarningStr = strings.TrimSpace(keepWarningStr)

		keepCriticalStr, _ := cmd.Flags().GetString("keep-critical")
		keepCriticalStr = strings.TrimSpace(keepCriticalStr)

		keepIncidentsStr, _ := cmd.Flags().GetString("keep-incidents")
		keepIncidentsStr = strings.TrimSpace(keepIncidentsStr)

		if keepStr == "" && keepHealthyStr == "" && keepWarningStr == "" && keepCriticalStr == "" && keepIncidentsStr == "" {
			return log.Client.Error("at least one --keep flag is required")
		} else if keepStr != "" && (keepHealthyStr != "" || keepWarningStr != "" || keepCriticalStr != "") {
			return log.Client.Error("--keep cannot be combined with --keep-healthy, --keep-warning, or --keep-critical")
		}

		parseDuration := func(flag, s string) (*time.Duration, error) {
			if s == "" {
				return nil, nil
			}
			d, err := utils.ParseDuration(s)
			if err != nil {
				return nil, fmt.Errorf("invalid %s — %s", flag, err)
			}
			return &d, nil
		}

		keep, err := parseDuration("--keep", keepStr)
		if err != nil {
			return log.Client.Error(err.Error())
		}

		keepHealthy, err := parseDuration("--keep-healthy", keepHealthyStr)
		if err != nil {
			return log.Client.Error(err.Error())
		}

		keepWarning, err := parseDuration("--keep-warning", keepWarningStr)
		if err != nil {
			return log.Client.Error(err.Error())
		}

		keepCritical, err := parseDuration("--keep-critical", keepCriticalStr)
		if err != nil {
			return log.Client.Error(err.Error())
		}

		keepIncidents, err := parseDuration("--keep-incidents", keepIncidentsStr)
		if err != nil {
			return log.Client.Error(err.Error())
		}

		cmd.SilenceUsage = true

		conn, err := ipc.Connect()
		if err != nil {
			return log.Client.Error(err.Error())
		}
		defer conn.Close()

		_, err = ipc.Send[dto.TruncateRequest, dto.TruncateResponse](conn, dto.TruncateRequest{
			Code:          code,
			Keep:          keep,
			KeepHealthy:   keepHealthy,
			KeepWarning:   keepWarning,
			KeepCritical:  keepCritical,
			KeepIncidents: keepIncidents,
		})
		if err != nil {
			return log.Client.Errorf("response error: %s", err)
		}

		return nil
	},
}

func init() {
	TruncateCmd.Flags().String("keep", "", "Delete samples and runs older than this duration")
	TruncateCmd.Flags().String("keep-healthy", "", "Retention duration for healthy samples and runs")
	TruncateCmd.Flags().String("keep-warning", "", "Retention duration for warning samples and runs")
	TruncateCmd.Flags().String("keep-critical", "", "Retention duration for critical samples and runs")
	TruncateCmd.Flags().String("keep-incidents", "", "Delete resolved incidents older than this duration")
}
