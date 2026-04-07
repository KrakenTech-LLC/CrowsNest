package dehashed

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"crowsnest/internal/files"
	"gopkg.in/yaml.v3"
)

const dataWellsEndpoint = "https://api.dehashed.com/data-wells"

type DataWellsRequest struct {
	Count int
	Page  int
	Sort  string
}

type DataWellsResponse struct {
	NextPage  bool       `json:"next_page" xml:"next_page" yaml:"next_page"`
	Total     int        `json:"total" xml:"total" yaml:"total"`
	DataWells []DataWell `json:"data_wells" xml:"data_wells" yaml:"data_wells"`
}

type DataWell struct {
	Data        string `json:"data" xml:"data" yaml:"data"`
	Date        string `json:"date" xml:"date" yaml:"date"`
	Description string `json:"description" xml:"description" yaml:"description"`
	Name        string `json:"name" xml:"name" yaml:"name"`
	Records     int    `json:"records" xml:"records" yaml:"records"`
	IsSensitive bool   `json:"is_sensitive" xml:"is_sensitive" yaml:"is_sensitive"`
}

func (dcv2 *DehashedClientV2) DataWells(request DataWellsRequest) (DataWellsResponse, error) {
	var dataWells DataWellsResponse

	endpoint, err := dataWellsURL(request)
	if err != nil {
		return dataWells, err
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return dataWells, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return dataWells, err
	}
	if res == nil {
		return dataWells, errors.New("response was nil")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return dataWells, err
	}

	if res.StatusCode != http.StatusOK {
		return dataWells, fmt.Errorf("data wells request failed: status=%d body=%s", res.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &dataWells); err != nil {
		return dataWells, err
	}

	return dataWells, nil
}

func dataWellsURL(request DataWellsRequest) (string, error) {
	if request.Page <= 0 {
		return "", errors.New("page must be 1 or greater")
	}
	if request.Count != 20 && request.Count != 50 {
		return "", errors.New("count must be 20 or 50")
	}
	if request.Sort != "" && !validDataWellsSort(request.Sort) {
		return "", fmt.Errorf("invalid sort %q; use added, name, date, records, optionally suffixed with -ASC or -DESC", request.Sort)
	}

	values := url.Values{}
	values.Set("page", strconv.Itoa(request.Page))
	values.Set("count", strconv.Itoa(request.Count))
	if request.Sort != "" {
		values.Set("sort", request.Sort)
	}

	return dataWellsEndpoint + "?" + values.Encode(), nil
}

func validDataWellsSort(sortValue string) bool {
	sortValue = strings.ToLower(strings.TrimSpace(sortValue))
	field := sortValue
	if before, _, ok := strings.Cut(sortValue, "-"); ok {
		field = before
	}

	switch field {
	case "added", "name", "date", "records":
		return strings.HasSuffix(sortValue, "-asc") || strings.HasSuffix(sortValue, "-desc") || !strings.Contains(sortValue, "-")
	default:
		return false
	}
}

func WriteDataWellsToFile(dataWells DataWellsResponse, outputFile string, fileType files.FileType) error {
	var data []byte
	var err error

	switch fileType {
	case files.JSON:
		data, err = json.MarshalIndent(dataWells, "", "  ")
	case files.XML:
		data, err = xml.MarshalIndent(dataWells, "", "  ")
	case files.YAML:
		data, err = yaml.Marshal(dataWells)
	case files.TEXT:
		data = []byte(dataWells.String())
	case files.GREPPABLE:
		var outStrings []string
		for _, well := range dataWells.DataWells {
			outStrings = append(outStrings, dataWellGreppable(well)+"\n")
		}
		data = []byte(strings.Join(outStrings, ""))
	default:
		return errors.New("unsupported file type")
	}

	if err != nil {
		return err
	}

	return os.WriteFile(outputFile+fileType.Extension(), data, 0644)
}

func (dwr DataWellsResponse) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Total: %d\nNext Page: %t\n\n", dwr.Total, dwr.NextPage)
	for _, well := range dwr.DataWells {
		fmt.Fprintf(&b, "Name: %s\nDate: %s\nRecords: %d\nSensitive: %t\nData: %s\nDescription: %s\n\n",
			well.Name,
			well.Date,
			well.Records,
			well.IsSensitive,
			well.Data,
			well.Description,
		)
	}
	return b.String()
}

func dataWellGreppable(well DataWell) string {
	fields := []string{
		"name=" + cleanGreppableValue(well.Name),
		"date=" + cleanGreppableValue(well.Date),
		"records=" + strconv.Itoa(well.Records),
		"is_sensitive=" + strconv.FormatBool(well.IsSensitive),
		"data=" + cleanGreppableValue(well.Data),
		"description=" + cleanGreppableValue(well.Description),
	}
	return strings.Join(fields, "\t")
}

func cleanGreppableValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	return strings.TrimSpace(value)
}
