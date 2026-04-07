package cmd

import (
	"crowsnest/internal/badger"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"strings"
)

var (
	// Global Flags
	debugGlobal bool

	// rootCmd is the base command for the CLI.
	rootCmd = &cobra.Command{
		Use:   "crowsnest",
		Short: `CrowsNest is a cli tool for querying the common OSINT api's.`,
		Long: fmt.Sprintf(
			"%s\n",
			`
    ╔═╗┬─┐┌─┐┬ ┬┌─┐╔╗╔┌─┐┌─┐┌┬┐
    ║  ├┬┘│ ││││└─┐║║║├┤ └─┐ │
    ╚═╝┴└─└─┘└┴┘└─┘╝╚╝└─┘└─┘ ┴

   Crow’s Nest OSINT Recon Suite
 ⚓ A KrakenTech Intelligence Tool
`,
		),
		Version: "v1.2.1",
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal("execute_root_command",
			zap.String("message", "failed to execute root command"),
			zap.Error(err),
		)
		fmt.Printf("[!] %v", err)
		os.Exit(1)
	}
}

func init() {
	// Hide the default help command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&debugGlobal, "debug", false, "Show debug information")

	// Add subcommands
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(setLocalDb)
	rootCmd.AddCommand(buyMeCoffeeCmd)

	setCmd.AddCommand(setDehashedKeyCmd)
	setCmd.AddCommand(setHunterKeyCmd)
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set CrowsNest configuration values",
	Long:  "Set CrowsNest configuration values such as API keys.",
}

// Command to set API key
var setDehashedKeyCmd = &cobra.Command{
	Use:   "dehashed [key]",
	Short: "Set and store Dehashed.com API key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		// Store key in badger DB
		err := storeDehashedApiKey(key)
		if err != nil {
			fmt.Printf("Error storing Dehashed API key: %v\n", err)
			return
		}
		fmt.Println("API key stored successfully")
	},
}

var setHunterKeyCmd = &cobra.Command{
	Use:   "hunter [key]",
	Short: "Set and store Hunter.io API key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		// Store key in badger DB
		err := storeHunterApiKey(key)
		if err != nil {
			fmt.Printf("Error storing Hunter API key: %v\n", err)
			return
		}
		fmt.Println("API key stored successfully")
	},
}

var setLocalDb = &cobra.Command{
	Use:   "local-db [true|false]",
	Short: "Set crowsnest to use a local database path instead of the default path",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var useLocalDatabase bool

		useLocal := strings.ToLower(args[0])

		if useLocal == "true" {
			useLocalDatabase = true
		} else if useLocal == "false" {
			useLocalDatabase = false
		} else {
			fmt.Println("Invalid argument. Please use 'true' or 'false'.")
			return
		}

		// Store useLocal in badger DB
		err := badger.StoreUseLocalDB(useLocalDatabase)
		if err != nil {
			fmt.Printf("Error storing local database useLocal: %v\n", err)
			return
		}
		fmt.Println("Local database useLocal stored successfully")
	},
}

var buyMeCoffeeCmd = &cobra.Command{
	Use:   "coffee",
	Short: "Support the project by buying a coffee",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(color.HiRedString("                                                                                   ;)(; "))
		fmt.Println(color.HiCyanString("                   We Hope You Enjoy Our Product                                  :----:"))
		fmt.Println(color.HiCyanString("                                                                                 C|====|"))
		fmt.Println(color.HiCyanString("                                                                                  |    |"))
		fmt.Print(color.HiGreenString(" Support the project by buying a coffee: "))
		fmt.Print(color.BlueString("https://buymeacoffee.com/ehosinskiz      "))
		fmt.Println(color.HiCyanString("`----'"))
	},
}

// Helper functions to store API credentials
func storeDehashedApiKey(key string) error {
	err := badger.StoreDehashedKey(key)
	if err != nil {
		fmt.Printf("Error storing API key: %v\n", err)
		return err
	}
	return nil
}

func storeHunterApiKey(key string) error {
	err := badger.StoreHunterKey(key)
	if err != nil {
		fmt.Printf("Error storing API key: %v\n", err)
		return err
	}
	return nil
}
