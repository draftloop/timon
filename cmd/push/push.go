package cmdpush

import (
	"github.com/spf13/cobra"
	"timon/cmd/push/job"
)

var PushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push something.",
}

func init() {
	PushCmd.AddCommand(PushIncidentCmd)
	PushCmd.AddCommand(PushProbeCmd)
	PushCmd.AddCommand(cmdpushjob.PushJobCmd)
}
