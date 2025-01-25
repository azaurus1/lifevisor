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
	"github.com/spf13/viper"
)

// this will be the scheduled task

var syncCmd = &cobra.Command{
	Use:   "sync [dbtype] [source-path] [connection-string] [interval]",
	Short: "Sync activity watch data by interval",
	Args:  cobra.MaximumNArgs(4), // Four arguments: dbtype, source-path, connection-string, interval
	Run: func(cmd *cobra.Command, args []string) {
		var dbType string
		var sourcePath string
		var connString string
		var intervalString string

		// Config
		configFile, _ := cmd.Flags().GetString("config")
		if configFile != "" {
			// using config
			viper.SetConfigFile(configFile)
			err := viper.ReadInConfig()
			if err != nil {
				log.Println("error reading config: ", err)
			}

			dbType = viper.GetString("dbType")
			sourcePath = viper.GetString("sourcePath")
			connString = viper.GetString("connString")
			intervalString = viper.GetString("interval")
		}

		interval, err := strconv.Atoi(intervalString)
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
	syncCmd.Flags().String("db-type", "", "Database type (optional)")
	syncCmd.Flags().String("source-path", "", "Source path (optional)")
	syncCmd.Flags().String("conn-string", "", "Connection string (optional)")
	syncCmd.Flags().String("interval", "", "Sync Interval (optional)")
	syncCmd.Flags().String("config", "", "Path to the configuration file (optional)")
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
