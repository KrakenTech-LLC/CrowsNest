package dehashed

import (
	"bytes"
	"crowsnest/internal/debug"
	"crowsnest/internal/sqlite"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

type DehashedParameter string

const (
	Username       DehashedParameter = "username"
	Email          DehashedParameter = "email"
	Password       DehashedParameter = "password"
	HashedPassword DehashedParameter = "hashed_password"
	Name           DehashedParameter = "name"
	IpAddress      DehashedParameter = "ip_address"
	Domain         DehashedParameter = "domain"
	Vin            DehashedParameter = "vin"
	LicensePlate   DehashedParameter = "license_plate"
	Address        DehashedParameter = "address"
	Phone          DehashedParameter = "phone"
	Social         DehashedParameter = "social"
	CryptoAddress  DehashedParameter = "cryptocurrency_address"
)

func (dp DehashedParameter) GetArgumentString(arg string) string {
	return fmt.Sprintf("%s:%s", string(dp), arg)
}

type DehashedSearchRequest struct {
	ForcePlaintext bool   `json:"-"`
	Debug          bool   `json:"-"`
	Page           int    `json:"page"`
	Query          string `json:"query"`
	Size           int    `json:"size"`
	Wildcard       bool   `json:"wildcard"`
	Regex          bool   `json:"regex"`
	DeDupe         bool   `json:"de_dupe"`
}

func NewDehashedSearchRequest(page, size int, wildcard, regex, forcePlaintext, debug bool) *DehashedSearchRequest {
	return &DehashedSearchRequest{Page: page, Query: "", Size: size, Wildcard: wildcard, Regex: regex, DeDupe: true, ForcePlaintext: forcePlaintext, Debug: debug}
}

func (dsr *DehashedSearchRequest) buildQuery(query string) {
	if dsr.Debug {
		debug.PrintInfo(fmt.Sprintf("building query: %s", query))
	}
	// Ensure query is properly formatted
	query = strings.TrimSpace(query)

	// For regex queries, we need to ensure the regex pattern is properly escaped
	// and not enquoted, as that would break the regex pattern
	if dsr.Regex && !strings.HasPrefix(query, "\"") && !strings.HasSuffix(query, "\"") {
		// Don't add extra quotes for regex patterns
	} else if strings.Contains(query, " ") && !strings.HasPrefix(query, "\"") {
		query = fmt.Sprintf("\"%s\"", query)
	}

	if len(dsr.Query) > 0 {
		dsr.Query = fmt.Sprintf("%s&%s", strings.TrimSpace(dsr.Query), query)
	} else {
		dsr.Query = query
	}

	if dsr.Debug {
		debug.PrintInfo(fmt.Sprintf("query built: %s", dsr.Query))
	}
}

func (dsr *DehashedSearchRequest) AddUsernameQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Username.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddEmailQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Email.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddIpAddressQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(IpAddress.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddDomainQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Domain.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddPasswordQuery(query string) {
	if dsr.ForcePlaintext {
		query = enquoteSpaced(query)
		dsr.buildQuery(Password.GetArgumentString(query))
		return
	}
	hash := sha256.Sum256([]byte(query))
	query = hex.EncodeToString(hash[:])
	dsr.AddHashedPasswordQuery(query)
}

func (dsr *DehashedSearchRequest) AddVinQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Vin.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddLicensePlateQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(LicensePlate.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddAddressQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Address.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddPhoneQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Phone.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddSocialQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Social.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddCryptoAddressQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(CryptoAddress.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddHashedPasswordQuery(query string) {
	query = strings.TrimSpace(query)
	dsr.buildQuery(HashedPassword.GetArgumentString(query))
}

func (dsr *DehashedSearchRequest) AddNameQuery(query string) {
	query = enquoteSpaced(query)
	dsr.buildQuery(Name.GetArgumentString(query))
}

type DehashedClientV2 struct {
	apiKey  string
	results []sqlite.Result
	debug   bool
}

func NewDehashedClientV2(apiKey string, debug bool) *DehashedClientV2 {
	return &DehashedClientV2{apiKey: apiKey, debug: debug}
}

func (dcv2 *DehashedClientV2) Search(searchRequest DehashedSearchRequest) (int, int, error) {
	if dcv2.debug {
		debug.PrintInfo("preparing search request")
		zap.L().Info("v2_search_debug",
			zap.String("message", "preparing search request"),
		)
	}

	// Create a copy of the search request to avoid modifying the original
	requestCopy := searchRequest

	reqBody, _ := json.Marshal(requestCopy)

	if dcv2.debug {
		j := string(reqBody)
		jReq := fmt.Sprintf("Request Body: %s\n", j)
		debug.PrintJson(jReq)
		zap.L().Info("v2_search_debug",
			zap.String("message", jReq),
			zap.String("body", j),
		)
	}

	req, err := http.NewRequest("POST", "https://api.dehashed.com/v2/search", bytes.NewReader(reqBody))
	if err != nil {
		return -1, -1, err
	}

	if dcv2.debug {
		debug.PrintInfo("setting headers")
		zap.L().Info("v2_search_debug",
			zap.String("message", "setting headers"),
		)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Dehashed-Api-Key", dcv2.apiKey)

	if dcv2.debug {
		headers := redactedHeaders(req.Header)
		h := fmt.Sprintf("Headers: %v\n", headers)
		debug.PrintJson(h)
		zap.L().Info("v2_search_debug",
			zap.String("message", h),
			zap.String("headers", fmt.Sprintf("%v", headers)),
		)

		debug.PrintInfo("performing request")
		zap.L().Info("v2_search_debug",
			zap.String("message", "performing request"),
		)
	}

	res, err := http.DefaultClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		if dcv2.debug {
			debug.PrintInfo("failed to perform request")
			debug.PrintError(err)
		}
		zap.L().Error("v2_search",
			zap.String("message", "failed to perform request"),
			zap.Error(err),
		)
		return -1, -1, err
	}
	if res == nil {
		if dcv2.debug {
			debug.PrintInfo("response was nil")
		}
		zap.L().Error("v2_search",
			zap.String("message", "response was nil"),
		)
		return -1, -1, errors.New("response was nil")
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		zap.L().Error("v2_search",
			zap.String("message", "failed to read response body"),
			zap.Error(err),
		)
		return -1, -1, err
	}

	// Check for HTTP status code errors
	if res.StatusCode != 200 {
		if dcv2.debug {
			debug.PrintInfo("received error status code")
			debug.PrintJson(fmt.Sprintf("Status Code: %d\n", res.StatusCode))
			debug.PrintJson(fmt.Sprintf("Body: %s\n", string(b[:])))
		}

		dhErr := GetDehashedError(res.StatusCode)
		zap.L().Error("v2_search",
			zap.String("message", "received error status code"),
			zap.Int("status_code", res.StatusCode),
			zap.String("error", dhErr.Error()),
			zap.String("body_error", string(b)),
		)
		return -1, -1, &dhErr
	}

	var responseResults sqlite.DehashedResponse
	err = json.Unmarshal(b, &responseResults)
	if err != nil {
		if dcv2.debug {
			debug.PrintInfo("failed to unmarshal response body")
			debug.PrintError(err)
		}
		zap.L().Error("v2_search",
			zap.String("message", "failed to unmarshal response body"),
			zap.Error(err),
		)
		return -1, -1, err
	}

	if dcv2.debug {
		debug.PrintInfo("appending results")
		debug.PrintJson(fmt.Sprintf("Total Results: %d\n", responseResults.TotalResults))
		debug.PrintJson(fmt.Sprintf("Balance: %d\n", responseResults.Balance))
		debug.PrintJson(fmt.Sprintf("Entries: %d\n", len(responseResults.Entries)))
	}

	dcv2.results = append(dcv2.results, responseResults.Entries...)
	return len(responseResults.Entries), responseResults.Balance, nil
}

func (dcv2 *DehashedClientV2) GetResults() sqlite.DehashedResults {
	return sqlite.DehashedResults{Results: dcv2.results}
}

func (dcv2 *DehashedClientV2) GetTotalResults() int {
	return len(dcv2.results)
}

func enquoteSpaced(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, " ") {
		return fmt.Sprintf("\"%s\"", s)
	}
	return s
}

func redactedHeaders(headers http.Header) http.Header {
	redacted := headers.Clone()
	if redacted.Get("Dehashed-Api-Key") != "" {
		redacted.Set("Dehashed-Api-Key", "[REDACTED]")
	}
	return redacted
}
