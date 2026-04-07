package dehashed

import (
	"strings"
	"testing"

	"crowsnest/internal/sqlite"
)

func TestSetQueriesCapsSearchAtFiftyThousandResults(t *testing.T) {
	options := &sqlite.QueryOptions{
		MaxRecords:   75000,
		MaxRequests:  -1,
		StartingPage: 1,
	}

	dehasher := NewDehasher(options)

	if dehasher.maxResults != maxSearchResultsPerQuery {
		t.Fatalf("maxResults = %d, want %d", dehasher.maxResults, maxSearchResultsPerQuery)
	}
	if dehasher.options.MaxRecords != maxSearchResultsPerPage {
		t.Fatalf("page size = %d, want %d", dehasher.options.MaxRecords, maxSearchResultsPerPage)
	}
	if dehasher.options.MaxRequests != 5 {
		t.Fatalf("max requests = %d, want 5", dehasher.options.MaxRequests)
	}
}

func TestSetQueriesHonorsExplicitRequestLimit(t *testing.T) {
	options := &sqlite.QueryOptions{
		MaxRecords:   50000,
		MaxRequests:  1,
		StartingPage: 1,
	}

	dehasher := NewDehasher(options)

	if dehasher.maxResults != maxSearchResultsPerPage {
		t.Fatalf("maxResults = %d, want %d", dehasher.maxResults, maxSearchResultsPerPage)
	}
	if dehasher.options.MaxRecords != maxSearchResultsPerPage {
		t.Fatalf("page size = %d, want %d", dehasher.options.MaxRecords, maxSearchResultsPerPage)
	}
	if dehasher.options.MaxRequests != 1 {
		t.Fatalf("max requests = %d, want 1", dehasher.options.MaxRequests)
	}
}

func TestDataWellsURLDoesNotRequireAPIKey(t *testing.T) {
	got, err := dataWellsURL(DataWellsRequest{
		Count: 50,
		Page:  2,
		Sort:  "records-DESC",
	})
	if err != nil {
		t.Fatalf("dataWellsURL returned error: %v", err)
	}

	if !strings.HasPrefix(got, dataWellsEndpoint+"?") {
		t.Fatalf("url = %q, want prefix %q", got, dataWellsEndpoint+"?")
	}
	gotLower := strings.ToLower(got)
	if strings.Contains(gotLower, "api_key") || strings.Contains(gotLower, "dehashed-api-key") {
		t.Fatalf("url contains API key material: %q", got)
	}
	for _, want := range []string{"count=50", "page=2", "sort=records-DESC"} {
		if !strings.Contains(got, want) {
			t.Fatalf("url = %q, want %q", got, want)
		}
	}
}
