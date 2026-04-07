package sqlite

import "testing"

func TestGetTableAcceptsDisplayedTableNames(t *testing.T) {
	tests := map[string]Table{
		"dehashed": ResultsTable,
		"results":  ResultsTable,
		"users":    CredsTable,
		"creds":    CredsTable,
	}

	for input, want := range tests {
		if got := GetTable(input); got != want {
			t.Fatalf("GetTable(%q) = %v, want %v", input, got, want)
		}
	}
}
