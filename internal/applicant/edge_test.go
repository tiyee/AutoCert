package applicant

import (
	"context"
	"testing"

	"github.com/tiyee/AutoCert/internal/edge"
)

type fakeEdgeDNS struct{ created, deleted int }

func (f *fakeEdgeDNS) CreateTXT(context.Context, string, string) (edge.Record, error) {
	f.created++
	return edge.Record{SiteID: "zone-own", ID: "record-own"}, nil
}
func (f *fakeEdgeDNS) DeleteTXT(_ context.Context, r edge.Record) error {
	if r.SiteID != "zone-own" || r.ID != "record-own" {
		panic("wrong record deleted")
	}
	f.deleted++
	return nil
}
func TestEdgeProviderRetainsRecordHandle(t *testing.T) {
	t.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")
	f := &fakeEdgeDNS{}
	p := edgeProvider(f)
	if err := p.Present("example.invalid", "token", "authorization"); err != nil {
		t.Fatal(err)
	}
	if err := p.CleanUp("example.invalid", "token", "authorization"); err != nil {
		t.Fatal(err)
	}
	if err := p.CleanUp("example.invalid", "token", "authorization"); err != nil {
		t.Fatal(err)
	}
	if f.created != 1 || f.deleted != 1 {
		t.Fatal("edge cleanup not idempotent")
	}
	for _, platform := range []string{"esa", "teo"} {
		if GetApplicant(platform) == nil {
			t.Fatal("missing DNS platform")
		}
	}
}
