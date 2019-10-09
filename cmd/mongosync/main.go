package main

import (
	"log"
	"github.com/spf13/cobra"
	"github.com/FusionFoundation/efsn/cmd/mongosync/sync"
)

func init() {
	initCmd()
}

func initCmd() {
	rootCmd.PersistentFlags().Uint64VarP(&sync.StartBlock, "startblock", "s", 0, "start syncing from")
	rootCmd.PersistentFlags().StringVarP(&sync.Endpoint, "attach", "a", "", "ipc socket path")
	rootCmd.PersistentFlags().BoolVarP(&partial, "partial", "p", false, "sync txs from selected addresses")
	rootCmd.PersistentFlags().StringSliceVar(&sync.Myaddrs, "addresses", []string{}, "addresses")
}

var partial bool = false

var rootCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		if partial {
			sync.RegisterTxFilter(sync.TxFilter2)
		}
		sync.Sync()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
