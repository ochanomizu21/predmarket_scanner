package main

import (
	"log"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "predmarket-scanner",
		Short: "Prediction market arbitrage scanner",
		Long:  "A research tool to identify arbitrage opportunities across prediction markets",
	}

	rootCmd.AddCommand(FetchMarketsCmd)
	rootCmd.AddCommand(ScanCmd)
	rootCmd.AddCommand(ExportCmd)
	rootCmd.AddCommand(RecordCmd)
	rootCmd.AddCommand(FetchHistoryCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
