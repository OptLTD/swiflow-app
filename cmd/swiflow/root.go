package main

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "swiflow",
		Short: "Swiflow agent runtime",
	}
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: ./config.json or SWIFLOW_CONFIG)")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose (debug) logging")

	root.AddCommand(serveCmd())
	root.AddCommand(migrateCmd())
	return root
}
