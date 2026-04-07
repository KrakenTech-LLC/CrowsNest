package export

import (
	"crowsnest/internal/files"
	"crowsnest/internal/sqlite"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
	"strings"
	"time"
)

func WriteCredsToFile(creds []sqlite.User, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(creds, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(creds, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(creds)
	case files.TEXT:
		var outStrings []string
		for _, c := range creds {
			outStrings = append(outStrings, c.ToString()+"\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	case files.GREPPABLE:
		var outStrings []string
		for _, c := range creds {
			var fields []string
			fields = appendGreppableField(fields, "email", c.Email)
			fields = appendGreppableField(fields, "username", c.Username)
			fields = appendGreppableField(fields, "password", c.Password)
			outStrings = append(outStrings, strings.Join(fields, " ")+"\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}

func WriteToFile(results sqlite.DehashedResults, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	result := results.Results

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(result, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(result, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(result)
	case files.TEXT:
		var outStrings []string
		for _, r := range result {
			out := fmt.Sprintf(
				"Id: %s\nEmail: %s\nIpAddress: %s\nUsername: %s\nPassword: %s\nHashedPassword: %s\nHashType: %s\nName: %s\nVin: %s\nAddress: %s\nPhone: %s\nDatabaseName: %s\n\n",
				r.DehashedId, r.Email, r.IpAddress, r.Username, r.Password, r.HashedPassword, r.HashType, r.Name, r.Vin, r.Address, r.Phone, r.DatabaseName)
			outStrings = append(outStrings, out)
		}
		data = []byte(strings.Join(outStrings, ""))
	case files.GREPPABLE:
		var outStrings []string
		for _, r := range result {
			outStrings = append(outStrings, dehashedResultGreppable(r)+"\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}

// WriteQueryResultsToFile writes query results to a file in the specified format
func WriteQueryResultsToFile(results []map[string]interface{}, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(results, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(results, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(results)
	case files.TEXT:
		var outStrings []string
		for _, r := range results {
			var rowStrings []string
			for k, v := range r {
				// Format the value to avoid array notation
				var valueStr string
				switch val := v.(type) {
				case []string:
					valueStr = strings.Join(val, ", ")
				case []interface{}:
					strSlice := make([]string, len(val))
					for i, item := range val {
						if item == nil {
							strSlice[i] = ""
						} else {
							strSlice[i] = fmt.Sprintf("%v", item)
						}
					}
					valueStr = strings.Join(strSlice, ", ")
				default:
					if v == nil {
						valueStr = ""
					} else {
						valueStr = fmt.Sprintf("%v", v)
					}
				}
				rowStrings = append(rowStrings, fmt.Sprintf("%s: %s", k, valueStr))
			}
			outStrings = append(outStrings, strings.Join(rowStrings, "\n")+"\n\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	case files.GREPPABLE:
		var outStrings []string
		for _, r := range results {
			keys := make([]string, 0, len(r))
			for k := range r {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			rowStrings := make([]string, 0, len(keys))
			for _, k := range keys {
				rowStrings = appendGreppableField(rowStrings, k, greppableAnyValue(r[k]))
			}
			outStrings = append(outStrings, strings.Join(rowStrings, " ")+"\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}

func dehashedResultGreppable(r sqlite.Result) string {
	var fields []string
	fields = appendGreppableField(fields, "id", r.DehashedId)
	fields = appendGreppableField(fields, "email", strings.Join(r.Email, ","))
	fields = appendGreppableField(fields, "ip_address", strings.Join(r.IpAddress, ","))
	fields = appendGreppableField(fields, "username", strings.Join(r.Username, ","))
	fields = appendGreppableField(fields, "password", strings.Join(r.Password, ","))
	fields = appendGreppableField(fields, "hashed_password", strings.Join(r.HashedPassword, ","))
	fields = appendGreppableField(fields, "hash_type", r.HashType)
	fields = appendGreppableField(fields, "name", strings.Join(r.Name, ","))
	fields = appendGreppableField(fields, "vin", strings.Join(r.Vin, ","))
	fields = appendGreppableField(fields, "license_plate", strings.Join(r.LicensePlate, ","))
	fields = appendGreppableField(fields, "url", strings.Join(r.Url, ","))
	fields = appendGreppableField(fields, "social", strings.Join(r.Social, ","))
	fields = appendGreppableField(fields, "cryptocurrency_address", strings.Join(r.CryptoCurrencyAddress, ","))
	fields = appendGreppableField(fields, "address", strings.Join(r.Address, ","))
	fields = appendGreppableField(fields, "phone", strings.Join(r.Phone, ","))
	fields = appendGreppableField(fields, "company", strings.Join(r.Company, ","))
	fields = appendGreppableField(fields, "database_name", r.DatabaseName)
	return strings.Join(fields, " ")
}

func greppableAnyValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []string:
		return greppableValue(strings.Join(v, ","))
	case []interface{}:
		values := make([]string, 0, len(v))
		for _, item := range v {
			values = append(values, fmt.Sprintf("%v", item))
		}
		return greppableValue(strings.Join(values, ","))
	case []byte:
		return greppableValue(string(v))
	default:
		return greppableValue(fmt.Sprintf("%v", v))
	}
}

func greppableValue(value string) string {
	return strings.Join(strings.Fields(value), "_")
}

func appendGreppableField(fields []string, key, value string) []string {
	value = greppableValue(value)
	if value == "" {
		return fields
	}
	return append(fields, fmt.Sprintf("%s=%s", key, value))
}

func WriteWhoIsHistoryToFile(results []sqlite.HistoryRecord, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(results, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(results, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(results)
	case files.TEXT:
		var outStrings []string
		for _, r := range results {
			outStrings = append(outStrings, r.String()+"\n\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}

func WriteWhoIsRecordToFile(record sqlite.WhoisRecord, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(record, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(record, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(record)
	case files.TEXT:
		data = []byte(record.String())
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}

func WriteSubdomainsToFile(records []sqlite.SubdomainRecord, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(records, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(records, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(records)
	case files.TEXT:
		var outStrings []string
		for _, r := range records {
			out := fmt.Sprintf(
				"Domain: %s\nFirst Seen: %s\nLast Seen: %s\n\n",
				r.Domain, time.Unix(r.FirstSeen, 0).String(), time.Unix(r.LastSeen, 0).String())
			outStrings = append(outStrings, out)
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}

func WriteIPLookupToFile(records []sqlite.LookupResult, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(records, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(records, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(records)
	case files.TEXT:
		var outStrings []string
		for _, r := range records {
			out := fmt.Sprintf(
				"Name: %s\nSearch Term: %s\nFirst Seen: %s\nLast Visit: %s\nType: %s\n\n",
				r.Name, r.SearchTerm, time.Unix(r.FirstSeen, 0).String(), time.Unix(r.LastVisit, 0).String(), r.Type)
			outStrings = append(outStrings, out)
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s.%s", outputFile, fileType.String())
	return os.WriteFile(filePath, data, 0644)
}
