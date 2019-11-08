package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/spf13/cobra"
	"github.com/FusionFoundation/efsn/cmd/mongosync/sync"
	"github.com/FusionFoundation/efsn/mongodb"
)

func init() {
	initCmd()
}

func initCmd() {
	rootCmd.PersistentFlags().Uint64VarP(&sync.StartBlock, "startblock", "s", 0, "start syncing from")
	rootCmd.PersistentFlags().StringVarP(&sync.Endpoint, "attach", "a", "", "ipc socket path")
	rootCmd.PersistentFlags().StringVar(&sync.IpcDataDir, "ipcdatadir", "./ipccli", "ipc socket path")
	rootCmd.PersistentFlags().StringVar(&sync.IpcDocRoot, "ipcdocroot", "./ipccli", "ipc socket path")
	rootCmd.PersistentFlags().StringVar(&mongodb.MongoIP, "mongo", "0.0.0.0:27017", "mongoip")
	rootCmd.PersistentFlags().StringVar(&mongodb.MgoUser, "mgouser", "", "mongo user")
	rootCmd.PersistentFlags().StringVar(&mongodb.MgoPwd, "mgopwd", "", "mongo password")
	//rootCmd.PersistentFlags().BoolVarP(&partial, "partial", "p", false, "sync txs from selected addresses")
	//rootCmd.PersistentFlags().StringSliceVar(&sync.Myaddrs, "addresses", []string{}, "addresses")
}

var partial bool = false

var rootCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		sync.InitSync()
		if partial {
			sync.RegisterTxFilter(sync.TxFilter2)
		}
		sync.Sync()
	},
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				sync.StopSync()
				os.Exit(0)
			}
		}
	}()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
