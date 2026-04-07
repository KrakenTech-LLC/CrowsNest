package dehashed

import (
	"crowsnest/internal/debug"
	"crowsnest/internal/export"
	"crowsnest/internal/pretty"
	"crowsnest/internal/sqlite"
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
)

const (
	maxSearchResultsPerPage  = 10000
	maxSearchResultsPerQuery = 50000
)

// Dehasher is a struct for querying the Dehashed API
type Dehasher struct {
	options    sqlite.QueryOptions
	nextPage   int
	debug      bool
	balance    int
	maxResults int
	request    *DehashedSearchRequest
	client     *DehashedClientV2
}

// NewDehasher creates a new Dehasher
func NewDehasher(options *sqlite.QueryOptions) *Dehasher {
	dh := &Dehasher{
		options:  *options,
		nextPage: options.StartingPage + 1,
		debug:    options.Debug,
		balance:  0,
	}
	dh.setQueries()
	dh.request = NewDehashedSearchRequest(dh.options.StartingPage, dh.options.MaxRecords, dh.options.WildcardMatch, dh.options.RegexMatch, false, options.Debug)
	dh.buildRequest()
	return dh
}

// SetClientCredentials sets the client credentials for the dehasher
func (dh *Dehasher) SetClientCredentials(key string) {
	dh.client = NewDehashedClientV2(key, dh.debug)
}

func (dh *Dehasher) getNextPage() int {
	if dh.debug {
		debug.PrintInfo(fmt.Sprintf("getting next page: %d", dh.nextPage))
	}
	nextPage := dh.nextPage
	dh.nextPage += 1
	return nextPage
}

// setQueries sets the number of queries to make based on the number of records and requests
func (dh *Dehasher) setQueries() {
	if dh.debug {
		debug.PrintInfo("setting queries")
	}

	if dh.options.MaxRequests == 0 {
		zap.L().Error("max requests cannot be zero")
		fmt.Println("[!] Max Requests cannot be zero")
		os.Exit(1)
	}

	requestedMaxResults := dh.options.MaxRecords
	if requestedMaxResults <= 0 {
		requestedMaxResults = maxSearchResultsPerQuery
	}
	if requestedMaxResults > maxSearchResultsPerQuery {
		requestedMaxResults = maxSearchResultsPerQuery
	}

	pageSize := requestedMaxResults
	if pageSize > maxSearchResultsPerPage {
		pageSize = maxSearchResultsPerPage
	}

	numQueries := (requestedMaxResults + pageSize - 1) / pageSize
	if dh.options.MaxRequests > 0 && dh.options.MaxRequests < numQueries {
		numQueries = dh.options.MaxRequests
	}

	dh.maxResults = requestedMaxResults
	if requestLimit := numQueries * pageSize; requestLimit < dh.maxResults {
		dh.maxResults = requestLimit
	}

	dh.options.MaxRecords = pageSize
	dh.options.MaxRequests = numQueries

	zap.L().Info("dehashed_search_pagination",
		zap.Int("max_results", dh.maxResults),
		zap.Int("page_size", dh.options.MaxRecords),
		zap.Int("max_requests", dh.options.MaxRequests),
	)

	if dh.debug {
		debug.PrintInfo(fmt.Sprintf("setting max requests: %d", numQueries))
		debug.PrintInfo(fmt.Sprintf("setting page size: %d", dh.options.MaxRecords))
		debug.PrintInfo(fmt.Sprintf("setting max results: %d", dh.maxResults))
	}

	fmt.Printf("Making %d Requests for up to %d Records (%d per request)\n", dh.options.MaxRequests, dh.maxResults, dh.options.MaxRecords)
}

// Start starts the querying process
func (dh *Dehasher) Start() {
	fmt.Printf("[*] Querying Dehashed API...\n")
	for i := 0; i < dh.options.MaxRequests; i++ {
		fmt.Printf("   [*] Performing Request...\n")
		count, balance, err := dh.client.Search(*dh.request)
		if err != nil {
			if dh.debug {
				debug.PrintInfo("error performing request")
				debug.PrintError(err)
			}

			// Check if it's a DehashError
			if dhErr, ok := err.(*DehashError); ok {
				fmt.Printf("      [!] Dehashed API Error: %s (Code: %d)\n", dhErr.Message, dhErr.Code)
				zap.L().Error("dehashed_api_error",
					zap.String("message", dhErr.Message),
					zap.Int("code", dhErr.Code),
				)
			} else {
				fmt.Printf("   [!] Error performing request: %v\n", err)
				zap.L().Error("request_error",
					zap.String("message", "failed to perform request"),
					zap.Error(err),
				)
			}

			if len(dh.client.results) > 0 {
				fmt.Printf("   [!] Partial results retrieved. Storing Results...\n")
				err := sqlite.StoreDehashedResults(dh.client.GetResults())
				if err != nil {
					zap.L().Error("store_results",
						zap.String("message", "failed to store results"),
						zap.Error(err),
					)
					fmt.Printf("   [!] Error storing results: %v\n", err)
				}
			}
			dh.parseResults()
			os.Exit(-1)
		}

		dh.balance = balance

		if count < dh.options.MaxRecords {
			fmt.Printf("      [+] Retrieved %d records\n", count)
			fmt.Printf("      [-] Not enough entries, ending queries\n")
			break
		} else {
			fmt.Printf("      [+] Retrieved %d records\n", count)
		}

		if dh.options.PrintBalance {
			fmt.Printf("      [*] Balance: %d\n", balance)
		}

		dh.request.Page = dh.getNextPage()
	}

	dh.parseResults()
}

// buildRequest constructs the query map
func (dh *Dehasher) buildRequest() {
	if len(dh.options.UsernameQuery) > 0 {
		dh.request.AddUsernameQuery(dh.options.UsernameQuery)
	}
	if len(dh.options.EmailQuery) > 0 {
		dh.request.AddEmailQuery(dh.options.EmailQuery)
	}
	if len(dh.options.IpQuery) > 0 {
		dh.request.AddIpAddressQuery(dh.options.IpQuery)
	}
	if len(dh.options.HashQuery) > 0 {
		dh.request.AddHashedPasswordQuery(dh.options.HashQuery)
	}
	if len(dh.options.PassQuery) > 0 {
		dh.request.AddPasswordQuery(dh.options.PassQuery)
	}
	if len(dh.options.NameQuery) > 0 {
		dh.request.AddNameQuery(dh.options.NameQuery)
	}
	if len(dh.options.DomainQuery) > 0 {
		dh.request.AddDomainQuery(dh.options.DomainQuery)
	}
	if len(dh.options.VinQuery) > 0 {
		dh.request.AddVinQuery(dh.options.VinQuery)
	}
	if len(dh.options.LicensePlateQuery) > 0 {
		dh.request.AddLicensePlateQuery(dh.options.LicensePlateQuery)
	}
	if len(dh.options.AddressQuery) > 0 {
		dh.request.AddAddressQuery(dh.options.AddressQuery)
	}
	if len(dh.options.PhoneQuery) > 0 {
		dh.request.AddPhoneQuery(dh.options.PhoneQuery)
	}
	if len(dh.options.SocialQuery) > 0 {
		dh.request.AddSocialQuery(dh.options.SocialQuery)
	}
	if len(dh.options.CryptoAddressQuery) > 0 {
		dh.request.AddCryptoAddressQuery(dh.options.CryptoAddressQuery)
	}
}

// parseResults parses the results and writes them to a file
func (dh *Dehasher) parseResults() {
	zap.L().Info("extracting_credentials")
	results := dh.client.GetResults()
	if dh.maxResults > 0 && len(results.Results) > dh.maxResults {
		results.Results = results.Results[:dh.maxResults]
	}
	creds := results.ExtractUsers()
	fmt.Printf("   [+] Discovered %d Credentials\n", len(creds))
	err := sqlite.StoreUsers(creds)
	if err != nil {
		zap.L().Error("store_creds",
			zap.String("message", "failed to store creds"),
			zap.Error(err),
		)
	}
	zap.L().Info("creds_stored", zap.Int("count", len(creds)))

	zap.L().Info("storing_results")
	err = sqlite.StoreDehashedResults(results)
	if err != nil {
		zap.L().Error("store_results",
			zap.String("message", "failed to store results"),
			zap.Error(err),
		)
	}
	zap.L().Info("results_stored", zap.Int("count", len(results.Results)))

	if len(results.Results) > 0 {
		var (
			headers = []string{"Email", "Username", "Password"}
			rows    [][]string
		)

		fmt.Printf("   [*] Writing entries to file: %s.%s\n", dh.options.OutputFile, dh.options.OutputFormat.String())
		if !dh.options.CredsOnly {
			err := export.WriteToFile(results, dh.options.OutputFile, dh.options.OutputFormat)
			if err != nil {
				fmt.Printf("[!] Error Writing to file: %v      Outputting to terminal.\n", err)
				zap.L().Error("write_results",
					zap.String("message", "failed to write results to file"),
					zap.Error(err),
				)
			} else {
				fmt.Println("      [*] Success")
			}

			if dh.debug {
				debug.PrintInfo("printing results table")
			}

			headers = []string{"Email", "Username", "Password", "Phone", "Company"}
			if len(results.Results) > 50 {
				fmt.Println("   [-] Large number of results recovered, displaying first 50...")
				for i := 0; i < 50; i++ {
					r := results.Results[i]
					rows = append(rows, []string{
						strings.Join(r.Email, ", "),
						strings.Join(r.Username, ", "),
						strings.Join(r.Password, ", "),
						strings.Join(r.Phone, ", "),
						strings.Join(r.Company, ", ")})
				}
			} else {
				for _, r := range results.Results {
					rows = append(rows, []string{
						strings.Join(r.Email, ", "),
						strings.Join(r.Username, ", "),
						strings.Join(r.Password, ", "),
						strings.Join(r.Phone, ", "),
						strings.Join(r.Company, ", ")})
				}
			}

			// Print Table
			pretty.Table(headers, rows)
		} else {
			if dh.debug {
				debug.PrintInfo("extracting credentials")
			}
			creds := results.ExtractUsers()
			if dh.debug {
				debug.PrintInfo("writing credentials to file")
			}
			err := export.WriteCredsToFile(creds, dh.options.OutputFile, dh.options.OutputFormat)
			if err != nil {
				fmt.Printf("[!] Error Writing to file: %v\n   Outputting to terminal.", err)
				zap.L().Error("write_creds",
					zap.String("message", "failed to write creds to file"),
					zap.Error(err),
				)
			} else {
				fmt.Println("      [*] Success")
			}

			if dh.debug {
				debug.PrintInfo("printing credentials table")
			}

			headers = []string{"Email", "Username", "Password"}
			if len(creds) > 50 {
				fmt.Println("   [-] Large number of results recovered, displaying first 50...")
				for i := 0; i < 50; i++ {
					c := creds[i]
					rows = append(rows, []string{c.Email, c.Username, c.Password})
				}
			} else {
				for _, c := range creds {
					rows = append(rows, []string{c.Email, c.Username, c.Password})
				}
			}

			// Print Table
			pretty.Table(headers, rows)
		}
	} else {
		fmt.Println("   [-] No results found")
	}
}
