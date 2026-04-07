package export

import (
	"strings"
	"testing"

	"crowsnest/internal/sqlite"
)

func TestDehashedResultGreppableUsesSpaceSeparatedNonEmptyTokens(t *testing.T) {
	got := dehashedResultGreppable(sqlite.Result{
		DehashedId: "123",
		Name:       []string{"Hargrave Mall"},
		Address:    []string{"irving tx"},
		Url:        []string{"gdt.com", "GDT.COM"},
	})

	if strings.Contains(got, "\t") {
		t.Fatalf("greppable output contains tab: %q", got)
	}
	if strings.Contains(got, "vin=") {
		t.Fatalf("greppable output contains empty field: %q", got)
	}
	for _, want := range []string{"id=123", "name=Hargrave_Mall", "address=irving_tx", "url=gdt.com,GDT.COM"} {
		if !strings.Contains(got, want) {
			t.Fatalf("greppable output = %q, want token %q", got, want)
		}
	}
}
