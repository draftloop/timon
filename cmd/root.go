package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"timon/cmd/push"
	"timon/internal/log"
)

var RootCmd = &cobra.Command{
	Use:   "timon",
	Short: "Timon",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if verbose {
			log.SetLevel(log.LevelDebug, "")
		}
		cmd.SilenceErrors = log.CurrentLevel != log.LevelSilent // If Timon doesn't display the logs, Cobra at least displays the errors
		return nil
	},
}
var verbose bool

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	RootCmd.CompletionOptions.DisableDefaultCmd = true

	RootCmd.AddCommand(DaemonCmd)
	RootCmd.AddCommand(VersionCmd)

	RootCmd.AddCommand(DeleteCmd)
	RootCmd.AddCommand(cmdpush.PushCmd)
	RootCmd.AddCommand(AnnotateCmd)
	RootCmd.AddCommand(ResolveCmd)
	RootCmd.AddCommand(ShowCmd)
	RootCmd.AddCommand(StatusCmd)
	RootCmd.AddCommand(SummaryCmd)
	RootCmd.AddCommand(TruncateCmd)
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
