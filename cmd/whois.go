package cmd

import (
	"crowsnest/internal/debug"
	"crowsnest/internal/export"
	"crowsnest/internal/files"
	"crowsnest/internal/pretty"
	"crowsnest/internal/sqlite"
	"crowsnest/internal/whois"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
)

func init() {
	// Add whois subcommand to root command
	rootCmd.AddCommand(whoisCmd)

	// Add flags specific to whois command
	whoisCmd.Flags().StringVarP(&whoisDomain, "domain", "d", "", "Domain for WHOIS lookup, history search, and subdomain scan")
	whoisCmd.Flags().StringVarP(&whoisIPAddress, "ip", "i", "", "IP address for reverse IP lookup")
	whoisCmd.Flags().StringVarP(&whoisMXAddress, "mx", "m", "", "MX hostname for reverse MX lookup")
	whoisCmd.Flags().StringVarP(&whoisNSAddress, "ns", "n", "", "NS hostname for reverse NS lookup")
	whoisCmd.Flags().StringVarP(&whoisInclude, "include", "I", "", "Up to 4 Terms to include in reverse WHOIS search (comma-separated)")
	whoisCmd.Flags().StringVarP(&whoisExclude, "exclude", "E", "", "Up to 4 Terms to exclude in reverse WHOIS search (comma-separated)")
	whoisCmd.Flags().StringVarP(&whoisReverseType, "type", "t", "current", "Type of reverse WHOIS search ([default] current or historic)")
	whoisCmd.Flags().StringVarP(&whoisOutputFormat, "format", "f", "text", "Output format (text, json)")
	whoisCmd.Flags().StringVarP(&whoisOutputFile, "output", "o", "whois", "File to output results to including extension")
	whoisCmd.Flags().BoolVarP(&whoisShowCredits, "credits", "c", false, "Show remaining WHOIS credits")
	whoisCmd.Flags().BoolVarP(&whoisHistory, "history", "H", false, "Perform WHOIS history search [25 Credits]")
	whoisCmd.Flags().BoolVarP(&whoisSubdomainScan, "subdomains", "s", false, "Perform WHOIS subdomain scan")
}

var (
	// WHOIS command flags
	whoisDomain        string
	whoisIPAddress     string
	whoisMXAddress     string
	whoisNSAddress     string
	whoisInclude       string
	whoisExclude       string
	whoisReverseType   string
	whoisOutputFormat  string
	whoisOutputFile    string
	whoisShowCredits   bool
	whoisHistory       bool
	whoisSubdomainScan bool

	// WHOIS command
	whoisCmd = &cobra.Command{
		Use:   "whois",
		Short: "Dehashed WHOIS lookups and reverse WHOIS searches",
		Long:  `Perform WHOIS lookups, history searches, reverse WHOIS searches, IP lookups, MX lookups, NS lookups, and subdomain scans.`,
		Run: func(cmd *cobra.Command, args []string) {
			key := getDehashedApiKey()

			// Validate credentials
			if key == "" {
				fmt.Println("API key is required. Set the key with the \"set dehashed\" command. [crowsnest set dehashed <api_key>]")
				return
			}

			if debugGlobal {
				debug.PrintInfo("debug mode enabled")
				zap.L().Info("whois_debug",
					zap.String("message", "debug mode enabled"),
				)
			}

			if whoisOutputFile == "" {
				if debugGlobal {
					debug.PrintInfo("output file not specified, using default")
				}
				whoisOutputFile = "whois_" + time.Now().Format("05_04_05")
			}

			if whoisOutputFormat == "" {
				if debugGlobal {
					debug.PrintInfo("output format not specified, using default")
				}
				whoisOutputFormat = "json"
			}

			fType := files.GetFileType(whoisOutputFormat)
			if fType == files.UNKNOWN {
				fmt.Println("[!] Error: Invalid output format. Must be 'json', 'xml', 'yaml', or 'txt'.")
				return
			}
			if debugGlobal {
				debug.PrintInfo("using output format: " + whoisOutputFormat)
			}

			w := whois.NewWhoIs(key, debugGlobal)

			// Show credits if requested
			if whoisShowCredits {
				fmt.Println("[*] Getting WHOIS balance...")
				if whoisShowCredits {
					checkBalance(w)
				}
			}

			// Check if domain is provided for history and subdomain scan
			if whoisHistory || whoisSubdomainScan {
				if whoisDomain == "" {
					fmt.Println("Domain is required for history and subdomain scan.")
					return
				}
				if whoisShowCredits {
					checkBalance(w)
				}
			}

			// Determine which operation to perform based on flags
			if whoisDomain != "" {
				fmt.Println("[*] Performing WHOIS lookup...")

				if !whoisHistory && !whoisSubdomainScan {
					// Domain lookup
					result, err := w.WhoisSearch(whoisDomain)
					if err != nil {
						if debugGlobal {
							debug.PrintInfo("failed to perform whois search")
							debug.PrintError(err)
						}
						zap.L().Error("whois_search",
							zap.String("message", "failed to perform whois search"),
							zap.Error(err),
						)
						fmt.Printf("Error performing WHOIS lookup: %v\n", err)
						return
					}

					if whoisShowCredits {
						checkBalance(w)
					}

					// Fix the output format to use proper formatting
					fmt.Println("WHOIS Lookup Result:")

					// Store the record
					err = sqlite.StoreWhoisRecord(result)
					if err != nil {
						if debugGlobal {
							debug.PrintInfo("failed to store whois record")
							debug.PrintError(err)
						}
						zap.L().Error("store_whois_record",
							zap.String("message", "failed to store whois record"),
							zap.Error(err),
						)
						fmt.Printf("Error storing WHOIS record: %v\n", err)
						// Continue execution even if storage fails
					}

					// Pretty Print WhoIs Record
					pretty.WhoIsTree(whoisDomain, result)

					// Write WhoIs Record to file
					if len(result.DomainName) != 0 {
						fmt.Printf("[*] Writing WHOIS record to file: %s%s\n", whoisOutputFile, fType.Extension())
						err = export.WriteWhoIsRecordToFile(result, whoisOutputFile, fType)
					} else {
						if debugGlobal {
							debug.PrintInfo("no whois record to write to file")
						}
						zap.L().Info("write_whois_record",
							zap.String("message", "no whois record to write to file"),
						)
					}
				}

				if whoisHistory {
					filename := whoisOutputFile + "_history"
					fmt.Println("[*] Performing WHOIS history search...")
					// Perform history search
					historyRecords, err := w.WhoisHistory(whoisDomain)
					if err != nil {
						if debugGlobal {
							debug.PrintInfo("failed to perform whois history lookup")
							debug.PrintError(err)
						}
						zap.L().Error("whois_history",
							zap.String("message", "failed to perform whois history lookup"),
							zap.Error(err),
						)
						fmt.Printf("[!] Error performing WHOIS history lookup: %v\n", err)
					} else {
						if whoisShowCredits {
							checkBalance(w)
						}

						// Write history records to file if any
						if len(historyRecords) > 0 {
							fmt.Printf("[*] Records Found: %d\n", len(historyRecords))
							fmt.Printf("[*] WHOIS History being written to file: %s%s\n", whoisOutputFile, fType.Extension())
							writeErr := export.WriteWhoIsHistoryToFile(historyRecords, filename, fType)
							if writeErr != nil {
								if debugGlobal {
									debug.PrintInfo("failed to write whois history to file")
									debug.PrintError(writeErr)
								}
								zap.L().Error("write_whois_history",
									zap.String("message", "failed to write whois history to file"),
									zap.Error(writeErr),
								)
								fmt.Printf("[!] Error writing WHOIS history to file: %v\n", writeErr)
							}

							err = sqlite.StoreWhoisHistoryRecords(historyRecords)
							if err != nil {
								if debugGlobal {
									debug.PrintInfo("failed to store history record")
									debug.PrintError(err)
								}
								zap.L().Error("store_history_record",
									zap.String("message", "failed to store history record"),
									zap.Error(err),
								)
								fmt.Printf("Error storing WHOIS history record: %v\n", err)
							}

						} else {
							if debugGlobal {
								debug.PrintInfo("no whois history records to write to file")
							}
							zap.L().Info("write_whois_history",
								zap.String("message", "no whois history records to write to file"),
							)
						}
					}
				}

				// Perform subdomain scan
				if whoisSubdomainScan {
					filename := whoisOutputFile + "_subdomains"
					fmt.Println("[*] Performing WHOIS subdomain scan...")
					subdomains, err := w.WhoisSubdomainScan(whoisDomain)

					// Get credits
					if whoisShowCredits {
						checkBalance(w)
					}

					if err != nil {
						if debugGlobal {
							debug.PrintInfo("failed to perform subdomain scan")
							debug.PrintError(err)
						}
						zap.L().Error("whois_subdomain_scan",
							zap.String("message", "failed to perform subdomain scan"),
							zap.Error(err),
						)
						fmt.Printf("Error performing subdomain scan: %v\n", err)
					} else {
						// Store subdomains in subdomains table
						var subs []sqlite.Subdomain
						for _, s := range subdomains {
							subs = append(subs, sqlite.Subdomain{Domain: whoisDomain, Subdomain: s.Domain})
						}

						err = sqlite.StoreSubdomains(subs)
						if err != nil {
							if debugGlobal {
								debug.PrintInfo("failed to store subdomain record")
								debug.PrintError(err)
							}
							zap.L().Error("store_subdomain_record",
								zap.String("message", "failed to store subdomain record"),
								zap.Error(err),
							)
							fmt.Printf("Error storing subdomain record: %v\n", err)
						}

						// Write the subdomains to file if any
						if len(subdomains) > 0 {
							fmt.Printf("[*] Writing subdomains to file: %s%s\n", whoisOutputFile, fType.Extension())
							err = export.WriteSubdomainsToFile(subdomains, filename, fType)
							if err != nil {
								zap.L().Error("write_whois_subdomain",
									zap.String("message", "failed to write whois subdomain to file"),
									zap.Error(err),
								)
								fmt.Printf("Error writing WHOIS subdomain to file: %v\n", err)
							}

							var (
								headers = []string{"Domain", "First Seen", "Last Seen"}
								rows    [][]string
							)

							// Create the rows
							for _, r := range subdomains {
								rows = append(rows, []string{r.Domain, time.Unix(r.FirstSeen, 0).String(), time.Unix(r.LastSeen, 0).String()})
							}

							// Store the subdomains
							fmt.Println("Subdomain Scan:")
							pretty.Table(headers, rows)

						} else {
							fmt.Println("[!] No subdomains found")
							zap.L().Info("write_whois_subdomain",
								zap.String("message", "no whois subdomain records to write to file"),
								zap.String("domain", whoisDomain),
							)
						}
					}
				}

				return
			}

			if whoisIPAddress != "" {
				fmt.Println("[*] Performing reverse IP lookup...")
				// IP lookup
				result, err := w.WhoisIP(whoisIPAddress)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to perform ip lookup")
						debug.PrintError(err)
					}
					zap.L().Error("whois_ip",
						zap.String("message", "failed to perform ip lookup"),
						zap.Error(err),
					)
					fmt.Printf("Error performing IP lookup: %v\n", err)
					return
				}

				// Get credits
				if whoisShowCredits {
					checkBalance(w)
				}

				if len(result) == 0 {
					fmt.Println("[!] No results found")
					zap.L().Info("whois_ip",
						zap.String("message", "no results found"),
						zap.String("ip", whoisIPAddress),
					)
					return
				}

				// Write the results to file
				fmt.Printf("[*] Writing IP lookup results to file: %s%s\n", whoisOutputFile, fType.Extension())
				err = export.WriteIPLookupToFile(result, whoisOutputFile, fType)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to write ip lookup to file")
						debug.PrintError(err)
					}
					zap.L().Error("write_ip_lookup",
						zap.String("message", "failed to write ip lookup to file"),
						zap.Error(err),
					)
					fmt.Printf("Error writing IP lookup to file: %v\n", err)
				}

				// Pretty Print the JSON
				var (
					headers = []string{"Name", "Search Term", "First Seen", "Last Visit", "Type"}
					rows    [][]string
				)
				fmt.Println("IP Lookup Result:")

				for _, r := range result {
					rows = append(rows, []string{r.Name, r.SearchTerm, time.Unix(r.FirstSeen, 0).String(), time.Unix(r.LastVisit, 0).String(), r.Type})
				}

				pretty.Table(headers, rows)

				return
			}

			if whoisMXAddress != "" {
				fmt.Println("[*] Performing reverse MX lookup...")
				// MX lookup
				result, err := w.WhoisMX(whoisMXAddress)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to perform mx lookup")
						debug.PrintError(err)
					}
					zap.L().Error("whois_mx",
						zap.String("message", "failed to perform mx lookup"),
						zap.Error(err),
					)
					fmt.Printf("Error performing MX lookup: %v\n", err)
					return
				}

				// Get credits
				if whoisShowCredits {
					checkBalance(w)
				}

				if len(result) == 0 {
					fmt.Println("[!] No results found")
					zap.L().Info("whois_mx",
						zap.String("message", "no results found"),
						zap.String("mx", whoisMXAddress),
					)
					return
				}

				// Write the results to file
				fmt.Printf("[*] Writing MX lookup results to file: %s%s\n", whoisOutputFile, fType.Extension())
				err = export.WriteIPLookupToFile(result, whoisOutputFile, fType)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to write mx lookup to file")
						debug.PrintError(err)
					}
					zap.L().Error("write_mx_lookup",
						zap.String("message", "failed to write mx lookup to file"),
						zap.Error(err),
					)
					fmt.Printf("Error writing MX lookup to file: %v\n", err)
				}

				// Pretty Print the JSON
				var (
					headers = []string{"Name", "Search Term", "First Seen", "Last Visit", "Type"}
					rows    [][]string
				)
				fmt.Println("MX Lookup Result:")

				for _, r := range result {
					rows = append(rows, []string{r.Name, r.SearchTerm, time.Unix(r.FirstSeen, 0).String(), time.Unix(r.LastVisit, 0).String(), r.Type})
				}

				pretty.Table(headers, rows)

				return
			}

			if whoisNSAddress != "" {
				fmt.Println("[*] Performing reverse NS lookup...")
				// NS lookup
				result, err := w.WhoisNS(whoisNSAddress)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to perform ns lookup")
						debug.PrintError(err)
					}
					zap.L().Error("whois_ns",
						zap.String("message", "failed to perform ns lookup"),
						zap.Error(err),
					)
					fmt.Printf("Error performing NS lookup: %v\n", err)
					return
				}

				// Get credits
				if whoisShowCredits {
					checkBalance(w)
				}

				if len(result) == 0 {
					fmt.Println("[!] No results found")
					zap.L().Info("whois_ns",
						zap.String("message", "no results found"),
						zap.String("ns", whoisNSAddress),
					)
					return
				}

				// Write the results to file
				fmt.Printf("[*] Writing NS lookup results to file: %s%s\n", whoisOutputFile, fType.Extension())
				err = export.WriteIPLookupToFile(result, whoisOutputFile, fType)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to write ns lookup to file")
						debug.PrintError(err)
					}
					zap.L().Error("write_ns_lookup",
						zap.String("message", "failed to write ns lookup to file"),
						zap.Error(err),
					)
					fmt.Printf("Error writing NS lookup to file: %v\n", err)
				}

				// Pretty Print the JSON
				var (
					headers = []string{"Name", "Search Term", "First Seen", "Last Visit", "Type"}
					rows    [][]string
				)
				fmt.Println("NS Lookup Result:")

				for _, r := range result {
					rows = append(rows, []string{r.Name, r.SearchTerm, time.Unix(r.FirstSeen, 0).String(), time.Unix(r.LastVisit, 0).String(), r.Type})
				}

				pretty.Table(headers, rows)

				return
			}

			if whoisInclude != "" || whoisExclude != "" {
				if debugGlobal {
					debug.PrintInfo("performing reverse whois")
					debug.PrintInfo("include: " + whoisInclude)
					debug.PrintInfo("exclude: " + whoisExclude)
					debug.PrintInfo("reverse type: " + whoisReverseType)
				}
				// Reverse WHOIS
				includeTerms := []string{}
				if whoisInclude != "" {
					includeTerms = strings.Split(whoisInclude, ",")
					if len(includeTerms) > 4 {
						fmt.Println("[!] Error: Maximum of 4 include terms allowed.")
						return
					}
				}

				excludeTerms := []string{}
				if whoisExclude != "" {
					excludeTerms = strings.Split(whoisExclude, ",")
					if len(excludeTerms) > 4 {
						fmt.Println("[!] Error: Maximum of 4 exclude terms allowed.")
						return
					}
				}

				toLower := strings.ToLower(whoisReverseType)
				if toLower != "current" && toLower != "historic" {
					fmt.Println("[!] Error: Invalid reverse type. Must be 'current' or 'historic'.")
					return
				}

				fmt.Println("[*] Performing reverse WHOIS lookup...")
				result, err := w.ReverseWHOIS(includeTerms, excludeTerms, whoisReverseType)
				if err != nil {
					if debugGlobal {
						debug.PrintInfo("failed to perform reverse whois")
						debug.PrintError(err)
					}
					zap.L().Error("reverse_whois",
						zap.String("message", "failed to perform reverse whois"),
						zap.Error(err),
					)
					fmt.Printf("Error performing reverse WHOIS: %v\n", err)
					return
				}

				// Write to file
				if len(result.DomainsList) > 0 {
					fmt.Printf("[*] Writing reverse WHOIS results to file: %s%s\n", whoisOutputFile, fType.Extension())
					err = export.WriteIStringToFile(result, whoisOutputFile, fType)
					if err != nil {
						if debugGlobal {
							debug.PrintInfo("failed to write reverse whois to file")
							debug.PrintError(err)
						}
						zap.L().Error("write_reverse_whois",
							zap.String("message", "failed to write reverse whois to file"),
							zap.Error(err),
						)
						fmt.Printf("Error writing reverse WHOIS to file: %v\n", err)
					}

					fmt.Println("Reverse WHOIS Result:")
					fmt.Printf("Total Domains: %d\n", result.DomainsCount)

					var (
						headers = []string{"Domain"}
						rows    [][]string
					)

					for _, r := range result.DomainsList {
						rows = append(rows, []string{r})
					}

					pretty.Table(headers, rows)
				} else {
					fmt.Println("[!] No results found")
					zap.L().Info("reverse_whois",
						zap.String("message", "no results found"),
					)
				}

				if whoisShowCredits {
					checkBalance(w)
				}
				return
			}

			// If no specific operation was requested
			cmd.Help()
		},
	}
)

func checkBalance(w *whois.DehashedWhoIs) {
	balance, err := w.Balance()
	if err != nil {
		if debugGlobal {
			debug.PrintInfo("failed to get whois balance")
			debug.PrintError(err)
		}
		zap.L().Error("get_whois_credits",
			zap.String("message", "failed to get whois balance"),
			zap.Error(err),
		)
		fmt.Printf("Error getting WHOIS balance: %v\n", err)
	}
	fmt.Println("WHOIS Credits: ", balance)
	if balance == 0 {
		fmt.Println("[!] No WHOIS credits remaining.")
		os.Exit(0)
	}
}
