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
			outStrings = append(outStrings, fmt.Sprintf("email=%s\tusername=%s\tpassword=%s\n",
				greppableValue(c.Email), greppableValue(c.Username), greppableValue(c.Password)))
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
				rowStrings = append(rowStrings, fmt.Sprintf("%s=%s", k, greppableAnyValue(r[k])))
			}
			outStrings = append(outStrings, strings.Join(rowStrings, "\t")+"\n")
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
	fields := []string{
		"id=" + greppableValue(r.DehashedId),
		"email=" + greppableValue(strings.Join(r.Email, ",")),
		"ip_address=" + greppableValue(strings.Join(r.IpAddress, ",")),
		"username=" + greppableValue(strings.Join(r.Username, ",")),
		"password=" + greppableValue(strings.Join(r.Password, ",")),
		"hashed_password=" + greppableValue(strings.Join(r.HashedPassword, ",")),
		"hash_type=" + greppableValue(r.HashType),
		"name=" + greppableValue(strings.Join(r.Name, ",")),
		"vin=" + greppableValue(strings.Join(r.Vin, ",")),
		"license_plate=" + greppableValue(strings.Join(r.LicensePlate, ",")),
		"url=" + greppableValue(strings.Join(r.Url, ",")),
		"social=" + greppableValue(strings.Join(r.Social, ",")),
		"cryptocurrency_address=" + greppableValue(strings.Join(r.CryptoCurrencyAddress, ",")),
		"address=" + greppableValue(strings.Join(r.Address, ",")),
		"phone=" + greppableValue(strings.Join(r.Phone, ",")),
		"company=" + greppableValue(strings.Join(r.Company, ",")),
		"database_name=" + greppableValue(r.DatabaseName),
	}
	return strings.Join(fields, "\t")
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
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	return strings.TrimSpace(value)
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
