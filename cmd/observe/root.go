package observe

import (
	"fmt"
	"os"

	"github.com/pol-cova/observe/internal/assistant"
	"github.com/pol-cova/observe/internal/detect"
	"github.com/pol-cova/observe/internal/prometheus"
	"github.com/pol-cova/observe/internal/tui"
	"github.com/spf13/cobra"
)

var prometheusURL, loadCommand string

var rootCmd = &cobra.Command{
	Use:   "observe",
	Short: "A zero-config terminal observability dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(tui.Options{PrometheusURL: prometheusURL, LoadCommand: loadCommand})
	},
}

func Execute() {
	rootCmd.Flags().StringVarP(&prometheusURL, "prometheus", "p", "", "Prometheus server URL")
	rootCmd.Flags().StringVarP(&loadCommand, "load", "l", "", "load test command to run")
	rootCmd.AddCommand(initCmd, askCmd, presetsCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var initCmd = &cobra.Command{
	Use: "init", Short: "Scan this machine and suggest observability setup",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := detect.Scan()
		if err != nil {
			return err
		}
		fmt.Print(report.String())
		return nil
	},
}

var askCmd = &cobra.Command{
	Use: "ask <question>", Short: "Explain current server health in plain English", Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshot, err := tui.Snapshot()
		if err != nil {
			return err
		}
		fmt.Print(assistant.Answer(args, snapshot))
		return nil
	},
}

var presetsCmd = &cobra.Command{
	Use: "presets", Short: "List useful built-in PromQL queries",
	Run: func(cmd *cobra.Command, args []string) {
		for _, p := range prometheus.Presets {
			fmt.Printf("%-18s %s\n  %s\n", p.Name, p.Description, p.Query)
		}
	},
}
