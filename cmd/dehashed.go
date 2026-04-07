package cmd

import (
	"crowsnest/internal/badger"
	"crowsnest/internal/debug"
	"crowsnest/internal/dehashed"
	"crowsnest/internal/files"
	"crowsnest/internal/pretty"
	"crowsnest/internal/sqlite"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	// Add api command to root command
	rootCmd.AddCommand(dehashedCmd)
	dehashedCmd.AddCommand(dehashedDataWellsCmd)

	// Add flags specific to api command
	dehashedCmd.Flags().IntVarP(&maxRecords, "max-records", "m", 50000, "Maximum total records to return (max 50000)")
	dehashedCmd.Flags().IntVarP(&maxRequests, "max-requests", "r", -1, "Maximum number of requests to make")
	dehashedCmd.Flags().IntVarP(&startingPage, "starting-page", "s", 1, "Starting page for requests")
	dehashedCmd.Flags().BoolVarP(&printBalance, "print-balance", "b", false, "Print remaining balance after requests")
	dehashedCmd.Flags().BoolVarP(&regexMatch, "regex-match", "R", false, "Use regex matching on query fields")
	dehashedCmd.Flags().BoolVarP(&wildcardMatch, "wildcard-match", "W", false, "Use wildcard matching on query fields (Use ? to replace a single character, and * for multiple characters)")
	dehashedCmd.Flags().BoolVarP(&credsOnly, "creds-only", "C", false, "Return credentials only")
	dehashedCmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format (json, yaml, xml, txt, grep)")
	dehashedCmd.Flags().StringVarP(&outputFile, "output", "o", "query", "File to output results to without extension")
	dehashedCmd.Flags().StringVarP(&usernameQuery, "username", "U", "", "Username query")
	dehashedCmd.Flags().StringVarP(&emailQuery, "email-query", "E", "", "Email query")
	dehashedCmd.Flags().StringVarP(&ipQuery, "ip", "I", "", "IP address query")
	dehashedCmd.Flags().StringVarP(&domainQuery, "domain", "D", "", "Domain query")
	dehashedCmd.Flags().StringVarP(&passwordQuery, "password", "P", "", "Password query")
	dehashedCmd.Flags().StringVarP(&vinQuery, "vin", "V", "", "VIN query")
	dehashedCmd.Flags().StringVarP(&licensePlateQuery, "license", "L", "", "License plate query")
	dehashedCmd.Flags().StringVarP(&addressQuery, "address", "A", "", "Address query")
	dehashedCmd.Flags().StringVarP(&phoneQuery, "phone", "M", "", "Phone query")
	dehashedCmd.Flags().StringVarP(&socialQuery, "social", "S", "", "Social query")
	dehashedCmd.Flags().StringVarP(&cryptoCurrencyAddressQuery, "crypto", "B", "", "Crypto currency address query")
	dehashedCmd.Flags().StringVarP(&hashQuery, "hash", "Q", "", "Hashed password query")
	dehashedCmd.Flags().StringVarP(&nameQuery, "name", "N", "", "Name query")

	// Add mutually exclusive flags to wildcard match and regex match
	dehashedCmd.MarkFlagsMutuallyExclusive("regex-match", "wildcard-match")

	dehashedDataWellsCmd.Flags().IntVar(&dataWellsCount, "count", 20, "Number of data wells to return (20 or 50)")
	dehashedDataWellsCmd.Flags().IntVarP(&dataWellsPage, "page", "p", 1, "Data wells page to request")
	dehashedDataWellsCmd.Flags().StringVar(&dataWellsSort, "sort", "", "Sort data wells by added, name, date, or records; optionally suffix -ASC or -DESC")
	dehashedDataWellsCmd.Flags().StringVarP(&dataWellsOutputFormat, "format", "f", "json", "Output format (json, yaml, xml, txt, grep)")
	dehashedDataWellsCmd.Flags().StringVarP(&dataWellsOutputFile, "output", "o", "data_wells", "File to output data wells to without extension")
}

var (
	// Query command flags
	maxRecords                 int
	maxRequests                int
	startingPage               int
	credsOnly                  bool
	printBalance               bool
	regexMatch                 bool
	wildcardMatch              bool
	outputFormat               string
	outputFile                 string
	usernameQuery              string
	emailQuery                 string
	ipQuery                    string
	passwordQuery              string
	hashQuery                  string
	nameQuery                  string
	domainQuery                string
	vinQuery                   string
	licensePlateQuery          string
	addressQuery               string
	phoneQuery                 string
	socialQuery                string
	cryptoCurrencyAddressQuery string
	dataWellsCount             int
	dataWellsPage              int
	dataWellsSort              string
	dataWellsOutputFormat      string
	dataWellsOutputFile        string

	// Query command
	dehashedCmd = &cobra.Command{
		Use:   "dehashed",
		Short: "Query the Dehashed API",
		Long:  `Query the Dehashed API for emails, usernames, passwords, hashes, IP addresses, and names.`,
		Run: func(cmd *cobra.Command, args []string) {
			key := getDehashedApiKey()

			// Validate credentials
			if key == "" {
				fmt.Println("API key is required. Set the key with the \"set dehashed\" command. [crowsnest set dehashed <api_key>]")
				return
			}

			// Create new QueryOptions
			queryOptions := sqlite.NewQueryOptions(
				maxRecords,
				maxRequests,
				startingPage,
				outputFormat,
				outputFile,
				usernameQuery,
				emailQuery,
				ipQuery,
				passwordQuery,
				hashQuery,
				nameQuery,
				domainQuery,
				vinQuery,
				licensePlateQuery,
				addressQuery,
				phoneQuery,
				socialQuery,
				cryptoCurrencyAddressQuery,
				regexMatch,
				wildcardMatch,
				printBalance,
				credsOnly,
				debugGlobal,
			)

			// Create new Dehasher
			dehasher := dehashed.NewDehasher(queryOptions)
			dehasher.SetClientCredentials(
				key,
			)

			// Start querying
			dehasher.Start()
			fmt.Println("\n[*] Completing Process")

			// Store query options
			err := sqlite.StoreDehashedQueryOptions(queryOptions)
			if err != nil {
				if debugGlobal {
					debug.PrintInfo("failed to store query options")
					debug.PrintError(err)
				}
				zap.L().Error("store_query_options",
					zap.String("message", "failed to store query options"),
					zap.Error(err),
				)
				fmt.Printf("Error storing query options: %v\n", err)
			}
		},
	}

	dehashedDataWellsCmd = &cobra.Command{
		Use:   "data-wells",
		Short: "List DeHashed data wells",
		Long:  `List DeHashed data wells. This endpoint is free and does not require a DeHashed API key or subscription.`,
		Run: func(cmd *cobra.Command, args []string) {
			client := dehashed.NewDehashedClientV2("", debugGlobal)
			response, err := client.DataWells(dehashed.DataWellsRequest{
				Count: dataWellsCount,
				Page:  dataWellsPage,
				Sort:  dataWellsSort,
			})
			if err != nil {
				fmt.Printf("[!] Error querying data wells: %v\n", err)
				return
			}

			fType := files.GetFileType(dataWellsOutputFormat)
			if dataWellsOutputFile != "" {
				fmt.Printf("[*] Writing data wells to file: %s%s\n", dataWellsOutputFile, fType.Extension())
				if err := dehashed.WriteDataWellsToFile(response, dataWellsOutputFile, fType); err != nil {
					fmt.Printf("[!] Error writing data wells to file: %v\n", err)
					return
				}
			}

			fmt.Printf("[+] Retrieved %d data wells (total: %d, next page: %t)\n", len(response.DataWells), response.Total, response.NextPage)
			printDataWellsTable(response.DataWells)
		},
	}
)

// Helper functions to get stored API credentials
func getDehashedApiKey() string {
	return badger.GetDehashedKey()
}

func printDataWellsTable(dataWells []dehashed.DataWell) {
	headers := []string{"Name", "Date", "Records", "Sensitive", "Data"}
	rows := make([][]string, 0, len(dataWells))
	for _, well := range dataWells {
		rows = append(rows, []string{
			well.Name,
			well.Date,
			fmt.Sprintf("%d", well.Records),
			fmt.Sprintf("%t", well.IsSensitive),
			well.Data,
		})
	}
	pretty.Table(headers, rows)
}
