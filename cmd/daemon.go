package cmd

import (
	"github.com/spf13/cobra"
	"path/filepath"
	configDefaults "timon/internal/config"
	"timon/internal/config/daemon"
	"timon/internal/daemon/cron"
	"timon/internal/daemon/db"
	"timon/internal/daemon/models"
	ipc "timon/internal/ipc/daemon"
	"timon/internal/log"
)

var DaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the daemon.",
	RunE: func(cmd *cobra.Command, args []string) error {
		background, _ := cmd.Flags().GetBool("background")

		cmd.SilenceUsage = true

		err := configdaemon.LoadConfig()
		if err != nil {
			return log.Daemon.Error(err.Error())
		}

		config := configdaemon.GetConfig()

		logLevel := log.CurrentLevel
		if logLevel == log.LevelSilent {
			logLevel, _ = log.ParseLevel(config.Daemon.LogLevel)
		}
		if background && logLevel != log.LevelSilent {
			logFilePath := filepath.Join(configdaemon.GetConfig().Daemon.LogDir, "timon.log")
			log.SetLevel(logLevel, logFilePath)
			log.Daemon.Debugf("logging into %s", logFilePath)
		} else {
			log.SetLevel(logLevel, "")
		}
		cmd.SilenceErrors = log.CurrentLevel != log.LevelSilent // If Timon doesn't display the logs, Cobra at least displays the errors

		configPath := configdaemon.GetConfigPath()
		if configPath != "" {
			log.Daemon.Debugf("config loaded from: %s", configPath)
		} else {
			log.Daemon.Debug("no config file found, using defaults")
		}

		db.GetDB()

		go cron.Start()

		if err := ipc.CreateServer(configDefaults.DefaultSocketPath, func() {
			go models.FireWebhookEventTimonStarted()
		}); err != nil {
			return log.Daemon.Error(err.Error())
		}

		return nil
	},
}

func init() {
	DaemonCmd.Flags().Bool("background", false, "Run in background mode — write logs to file in addition to stdout")
}
