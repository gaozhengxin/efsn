package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	_, err := NewPIDFile("pid.file")
	if err != nil {
		log.Fatal("error to create the pid file")
	}
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

type PIDFile struct {
    path string
}

func processExists(pid string) bool {
    if _, err := os.Stat(filepath.Join("/proc", pid)); err == nil {
        return true
    }
    return false
}

func checkPIDFILEAlreadyExists(path string) error {
    if pidByte, err := ioutil.ReadFile(path); err == nil {
        pid := strings.TrimSpace(string(pidByte))
        if processExists(pid) {
            return fmt.Errorf("ensure the process:%s is not running pid file:%s", pid, path)
        }
    }
    return nil
}

func NewPIDFile(path string) (*PIDFile, error) {
    if err := checkPIDFILEAlreadyExists(path); err != nil {
        return nil, err
    }

    if err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755)); err != nil {
        return nil, err
    }
    if err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
        return nil, err
    }
    return &PIDFile{path: path}, nil
}

func (file PIDFile) Remove() error {
    return os.Remove(file.path)
}
