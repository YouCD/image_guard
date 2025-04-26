package cmd

import (
	"github.com/spf13/cobra"
	"github.com/youcd/toolkit/log"
	"image_guard/guard"
	"os"
)

//nolint:gochecknoglobals
var (
	Name       = "image_guard"
	schedule   string
	containers []string
)

//nolint:gochecknoinits
func init() {
	log.Init(true)
	log.SetLogLevel("info")
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVarP(&schedule, "schedule", "s", "*/1 * * * *", "cron schedule")
	rootCmd.PersistentFlags().StringSliceVarP(&containers, "containers", "c", containers, "watch containers")
}

// rootCmd represents the base command when called without any subcommands
//
var rootCmd = &cobra.Command{
	Use:   Name,
	Short: Name + "watch container image update",

	Run: func(_ *cobra.Command, _ []string) {
		guard.CronScheduling(schedule, containers)
		select {}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
