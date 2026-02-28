package cmdpushjob

import (
	"github.com/spf13/cobra"
)

var PushJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Push a job.",
}

func init() {
	PushJobCmd.AddCommand(PushJobStartCmd, PushJobSepCmd, PushJobEndCmd)
}
