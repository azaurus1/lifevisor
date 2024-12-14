package cmd

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/azaurus1/lifevisor/internal/direct"
	"github.com/azaurus1/lifevisor/internal/http"
	"github.com/spf13/cobra"
)

// this will be the scheduled task

var syncCmd = &cobra.Command{
	Use:   "sync [dbtype] [source-path] [connection-string] [interval]",
	Short: "Sync activity watch data by interval",
	Args:  cobra.ExactArgs(4), // Four arguments: dbtype, source-path, connection-string, interval
	Run: func(cmd *cobra.Command, args []string) {
		dbType := args[0]
		sourcePath := args[1]
		connString := args[2]
		interval, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatal("cannot convert interval to int: ", err)
		}

		// Determine the connection type based on the connString
		var isHTTP bool
		if strings.HasPrefix(connString, "http://") || strings.HasPrefix(connString, "https://") {
			isHTTP = true
		}

		// Call the Sync method
		err = Sync(dbType, sourcePath, connString, interval, isHTTP)
		if err != nil {
			cmd.PrintErrln("Error during sync:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func Sync(dbType, sourcePath, connString string, interval int, isHTTP bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if isHTTP {
		err := http.HttpSync(ctx, sourcePath, connString, interval)
		if err != nil {
			return err
		}
	} else {
		err := direct.DirectSync(ctx, dbType, sourcePath, connString, interval)
		if err != nil {
			return err
		}
	}

	return nil
}
