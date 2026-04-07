package cmd

import (
	"crowsnest/internal/debug"
	"crowsnest/internal/export"
	"crowsnest/internal/files"
	"crowsnest/internal/pretty"
	"crowsnest/internal/sqlite"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"strings"
)

func init() {
	// Add whois command to root command
	rootCmd.AddCommand(queryCmd)

	// Add flags specific to whois command
	queryCmd.Flags().StringVarP(&dbQueryTableName, "table", "t", "", "Table to query (dehashed, users, whois, subdomains, lookup, runs)")
	queryCmd.Flags().IntVarP(&dbQueryLimitRows, "limit", "l", 100, "Limit number of results")
	queryCmd.Flags().StringVarP(&dbQueryNotNull, "not-null", "n", "", "Filter for non-null values (comma-separated list, e.g., 'password,email')")
	queryCmd.Flags().StringVarP(&dbQueryColumns, "columns", "c", "", "Columns to display in output (comma-separated list, e.g., 'username,email,password')")
	queryCmd.Flags().StringVarP(&dbQueryUserQuery, "user-query", "q", "", "User query to execute")
	queryCmd.Flags().StringVarP(&dbQueryRawQuery, "raw-query", "r", "", "Raw SQL query to execute")
	queryCmd.Flags().BoolVarP(&dbQueryListAll, "list-all", "a", false, "List all tables and their columns")
	queryCmd.Flags().BoolVarP(&dbQueryExport, "export", "x", false, "Export results to file using --file and --format")
	queryCmd.Flags().StringVarP(&dbQueryFormat, "format", "f", "json", "Output format (json, yaml, xml, txt, grep)")
	queryCmd.Flags().StringVarP(&dbQueryFile, "file", "o", "query", "File to output results to when --export is set")

	// Add mutually exclusive flags to query and raw-query
	// Cannot use query and raw-query at the same time
	queryCmd.MarkFlagsMutuallyExclusive("user-query", "raw-query")
	// Raw query does not require a table
	queryCmd.MarkFlagsMutuallyExclusive("raw-query", "table")
	// List all columns does not require a query or raw-query
	queryCmd.MarkFlagsMutuallyExclusive("raw-query", "list-all")
}

var (
	dbQueryTableName string
	dbQueryLimitRows int
	dbQueryNotNull   string
	dbQueryColumns   string
	dbQueryUserQuery string
	dbQueryRawQuery  string
	dbQueryListAll   bool
	dbQueryExport    bool
	dbQueryFormat    string
	dbQueryFile      string

	queryCmd = &cobra.Command{
		Use:   "query",
		Short: "Query the database",
		Long: `Query the database for various information.
Use --export with --file and --format to write results to a file instead of displaying them.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If list-all flag is set, list all tables and columns
			if dbQueryListAll {
				listAvailableTables()
				return
			}

			// If Raw Query is set, execute it and return
			if dbQueryRawQuery != "" {
				fmt.Println("[*] Executing Raw Query...")
				rawDBQuery()
				return
			}

			// Validate table name
			if dbQueryTableName == "" {
				fmt.Println("[!] Error: Table name is required. Use -t or --table to specify a table.")
				fmt.Println("[*] Available tables: dehashed, users, whois, subdomains, lookup, runs")
				fmt.Println("[*] Use --list-all to see all tables and their columns.")
				return
			}

			if !isValidTable(dbQueryTableName) {
				fmt.Printf("[!] Error: Unknown table '%s'.\n", dbQueryTableName)
				fmt.Println("[*] Available tables: dehashed, users, whois, subdomains, lookup, runs")
				fmt.Println("[*] Use --list-all to see all tables and their columns.")
				return
			}

			// Validate columns if specified
			if dbQueryColumns != "" {
				columns := strings.Split(dbQueryColumns, ",")
				invalidColumns := validateColumns(dbQueryTableName, columns)
				if len(invalidColumns) > 0 {
					fmt.Printf("[!] Error: Invalid column(s) for table '%s': %s\n",
						dbQueryTableName, strings.Join(invalidColumns, ", "))
					fmt.Println("[*] Available columns for this table:")
					for i := 0; i < len(availableTables[dbQueryTableName]); i += 5 {
						end := i + 5
						if end > len(availableTables[dbQueryTableName]) {
							end = len(availableTables[dbQueryTableName])
						}
						fmt.Printf("    %s\n", strings.Join(availableTables[dbQueryTableName][i:end], ", "))
					}
					return
				}
			}

			// Validate not-null fields if specified
			if dbQueryNotNull != "" {
				notNullFields := strings.Split(dbQueryNotNull, ",")
				invalidFields := validateColumns(dbQueryTableName, notNullFields)
				if len(invalidFields) > 0 {
					fmt.Printf("[!] Error: Invalid not-null field(s) for table '%s': %s\n",
						dbQueryTableName, strings.Join(invalidFields, ", "))
					fmt.Println("[*] Available columns for this table:")
					for i := 0; i < len(availableTables[dbQueryTableName]); i += 5 {
						end := i + 5
						if end > len(availableTables[dbQueryTableName]) {
							end = len(availableTables[dbQueryTableName])
						}
						fmt.Printf("    %s\n", strings.Join(availableTables[dbQueryTableName][i:end], ", "))
					}
					return
				}
			}

			// Determine which table to query based on the tableTypeDBQuery parameter
			table := sqlite.GetTable(dbQueryTableName)
			if table == sqlite.UnknownTable {
				fmt.Printf("[!] Error: Unknown table type '%s'.\n", dbQueryTableName)
				fmt.Println("[*] Available tables: dehashed, users, whois, subdomains, lookup, runs")
				fmt.Println("[*] Use --list-all to see all tables and their columns.")
				return
			}

			fmt.Println("[*] Querying Database...")
			tableQuery(table)
		},
	}
)

func tableQuery(table sqlite.Table) {

	// Get the columns to query
	columns := []string{"*"}
	if dbQueryColumns != "" {
		columns = strings.Split(dbQueryColumns, ",")
	}

	// Get the not null fields
	notNullFields := []string{}
	if dbQueryNotNull != "" {
		notNullFields = strings.Split(dbQueryNotNull, ",")
	}

	// Get the user query
	userQuery := ""
	if dbQueryUserQuery != "" {
		userQuery = dbQueryUserQuery
	}

	// Get the limit
	limit := dbQueryLimitRows

	// Get the object for the table
	object := table.Object()

	// Check if object is nil (invalid table)
	if object == nil {
		fmt.Printf("[!] Error: Table '%s' is not valid or does not exist.\n", dbQueryTableName)
		return
	}

	// Query the database
	db := sqlite.GetDB()
	query := db.Model(object).Select(columns)
	if len(notNullFields) > 0 {
		for _, field := range notNullFields {
			query = query.Where(fmt.Sprintf("%s IS NOT NULL", field))
		}
	}
	if userQuery != "" {
		query = query.Where(userQuery)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	rows, err := query.Rows()
	if err != nil {
		zap.L().Error("db_query",
			zap.String("message", "failed to execute query"),
			zap.Error(err),
		)
		fmt.Printf("[!] Error executing query: %v\n", err)
		return
	}
	defer rows.Close()

	// Get the columns
	cols, err := rows.Columns()
	if err != nil {
		zap.L().Error("db_query",
			zap.String("message", "failed to get columns from query"),
			zap.Error(err),
		)
		fmt.Printf("[!] Error getting columns from query: %v\n", err)
		return
	}

	if dbQueryExport {
		fmt.Println("[*] Exporting results to file...")

		if debugGlobal {
			debug.PrintInfo("exporting results to file: " + dbQueryFile)
		}
		// Prepare data for export
		var results []map[string]interface{}

		// Process the rows
		for rows.Next() {
			values := make([]interface{}, len(cols))
			pointers := make([]interface{}, len(cols))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				zap.L().Error("export_query",
					zap.String("message", "failed to scan row from query"),
					zap.Error(err),
				)
				fmt.Printf("[!] Error scanning row from query: %v\n", err)
				return
			}

			// Create a map for this row
			rowMap := make(map[string]interface{})
			for i, col := range cols {
				val := values[i]
				rowMap[col] = val
			}

			results = append(results, rowMap)
		}

		// Export the results
		exportQueryResults(results)

		return
	}
	// Prepare data for pretty.Table
	headers := cols
	var tableRows [][]string

	// Process the rows
	for rows.Next() {
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			zap.L().Error("db_query",
				zap.String("message", "failed to scan row from query"),
				zap.Error(err),
			)
			fmt.Printf("[!] Error scanning row from query: %v\n", err)
			continue
		}

		// Convert row values to strings
		rowStrings := make([]string, len(values))
		for i, value := range values {
			if value == nil {
				rowStrings[i] = " "
			} else {
				// Check if the value is a slice or array
				switch v := value.(type) {
				case []string:
					// Join string slices with commas, no brackets
					rowStrings[i] = strings.Join(v, ", ")
				case []interface{}:
					// Convert interface slice to strings and join
					strSlice := make([]string, len(v))
					for j, item := range v {
						if item == nil {
							strSlice[j] = ""
						} else {
							strSlice[j] = fmt.Sprintf("%v", item)
						}
					}
					rowStrings[i] = strings.Join(strSlice, ", ")
				case string:
					// Handle JSON strings that might be arrays
					if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
						// Try to unmarshal JSON array
						var strArray []string
						if err := json.Unmarshal([]byte(v), &strArray); err == nil {
							rowStrings[i] = strings.Join(strArray, ", ")
						} else {
							rowStrings[i] = v
						}
					} else {
						rowStrings[i] = v
					}
				default:
					rowStrings[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		tableRows = append(tableRows, rowStrings)
	}

	// Display the table
	pretty.Table(headers, tableRows)
}

func rawDBQuery() {
	db := sqlite.GetDB()
	rows, err := db.Raw(dbQueryRawQuery).Rows()
	if err != nil {
		zap.L().Error("raw_query",
			zap.String("message", "failed to execute raw query"),
			zap.Error(err),
		)
		fmt.Printf("[!] Error executing raw query: %v\n", err)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		zap.L().Error("raw_query",
			zap.String("message", "failed to get columns from raw query"),
			zap.Error(err),
		)
		fmt.Printf("[!] Error getting columns from raw query: %v\n", err)
		return
	}

	if dbQueryExport {
		fmt.Println("[*] Exporting results to file...")

		if debugGlobal {
			debug.PrintInfo("exporting results to file: " + dbQueryFile)
		}

		// Prepare data for export
		var results []map[string]interface{}

		// Process the rows
		for rows.Next() {
			values := make([]interface{}, len(columns))
			pointers := make([]interface{}, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				zap.L().Error("export_raw_query",
					zap.String("message", "failed to scan row from raw query"),
					zap.Error(err),
				)
				fmt.Printf("[!] Error scanning row from raw query: %v\n", err)
				return
			}

			// Create a map for this row
			rowMap := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				rowMap[col] = val
			}

			results = append(results, rowMap)
		}

		// Export the results
		exportQueryResults(results)

		return
	}
	// Prepare data for pretty.Table
	headers := columns
	var tableRows [][]string

	// Process the rows
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			zap.L().Error("raw_query",
				zap.String("message", "failed to scan row from raw query"),
				zap.Error(err),
			)
			fmt.Printf("[!] Error scanning row from raw query: %v\n", err)
			continue
		}

		// Convert row values to strings
		rowStrings := make([]string, len(values))
		for i, value := range values {
			if value == nil {
				rowStrings[i] = " "
			} else {
				// Check if the value is a slice or array
				switch v := value.(type) {
				case []string:
					// Join string slices with commas, no brackets
					rowStrings[i] = strings.Join(v, ", ")
				case []interface{}:
					// Convert interface slice to strings and join
					strSlice := make([]string, len(v))
					for j, item := range v {
						if item == nil {
							strSlice[j] = ""
						} else {
							strSlice[j] = fmt.Sprintf("%v", item)
						}
					}
					rowStrings[i] = strings.Join(strSlice, ", ")
				case string:
					// Handle JSON strings that might be arrays
					if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
						// Try to unmarshal JSON array
						var strArray []string
						if err := json.Unmarshal([]byte(v), &strArray); err == nil {
							rowStrings[i] = strings.Join(strArray, ", ")
						} else {
							rowStrings[i] = v
						}
					} else {
						rowStrings[i] = v
					}
				default:
					rowStrings[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		tableRows = append(tableRows, rowStrings)
	}

	// Display the table
	pretty.Table(headers, tableRows)
}

// exportQueryResults exports the results to a file
func exportQueryResults(results []map[string]interface{}) {
	// Get file type
	fileType := files.GetFileType(dbQueryFormat)

	// Export results
	err := export.WriteQueryResultsToFile(results, dbQueryFile, fileType)
	if err != nil {
		zap.L().Error("export_results",
			zap.String("message", "failed to write to file"),
			zap.Error(err),
		)
		fmt.Printf("[!] Error writing to file: %v\n", err)
		return
	}

	fmt.Printf("[+] Exported %d records to file: %s%s\n", len(results), dbQueryFile, fileType.Extension())
}
