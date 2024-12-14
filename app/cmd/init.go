package cmd

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/azaurus1/lifevisor/internal/direct"
	lifevisorHttp "github.com/azaurus1/lifevisor/internal/http"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [dbtype] [source-path] [connection-string] [concurrency]",
	Short: "Run initial load of data to the specified database type",
	Args:  cobra.ExactArgs(4), // Four arguments: dbtype, source-path, connection-string, concurrency
	Run: func(cmd *cobra.Command, args []string) {
		dbType := args[0]
		sourcePath := args[1]
		connString := args[2]

		concurrency, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatal("error converting concurrency to int: ", err)
		}

		// Determine the connection type based on the connString
		var isHTTP bool
		if strings.HasPrefix(connString, "http://") || strings.HasPrefix(connString, "https://") {
			isHTTP = true
		}

		// Call the Initialize method
		err = Initialisation(dbType, sourcePath, connString, concurrency, isHTTP)
		if err != nil {
			cmd.PrintErrln("Error during initialization:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func Initialisation(dbType, sqlitePath, connString string, concurrency int, isHTTP bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if isHTTP {
		err := lifevisorHttp.HttpInitialisation(ctx, sqlitePath, connString, concurrency)
		if err != nil {
			return err
		}
	} else {
		err := direct.DirectInitialisation(ctx, dbType, sqlitePath, connString, concurrency)
		if err != nil {
			return err
		}
	}

	return nil
}
