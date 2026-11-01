package edge

import (
	"context"
	"testing"

	esa "github.com/alibabacloud-go/esa-20240910/v3/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type esaMock struct {
	esaAPI
	list    func(*esa.ListSitesRequest) *esa.ListSitesResponse
	create  func(*esa.CreateRecordRequest) *esa.CreateRecordResponse
	remove  func(*esa.DeleteRecordRequest) *esa.DeleteRecordResponse
	matches func(*esa.ListCertificatesByRecordRequest) *esa.ListCertificatesByRecordResponse
	get     func(*esa.GetCertificateRequest) *esa.GetCertificateResponse
	set     func(*esa.SetCertificateRequest) *esa.SetCertificateResponse
}

func (m *esaMock) ListSitesWithOptions(r *esa.ListSitesRequest, _ *dara.RuntimeOptions) (*esa.ListSitesResponse, error) {
	return m.list(r), nil
}
func (m *esaMock) CreateRecordWithOptions(r *esa.CreateRecordRequest, _ *dara.RuntimeOptions) (*esa.CreateRecordResponse, error) {
	return m.create(r), nil
}
func (m *esaMock) DeleteRecordWithOptions(r *esa.DeleteRecordRequest, _ *dara.RuntimeOptions) (*esa.DeleteRecordResponse, error) {
	return m.remove(r), nil
}
func (m *esaMock) ListCertificatesByRecordWithOptions(r *esa.ListCertificatesByRecordRequest, _ *dara.RuntimeOptions) (*esa.ListCertificatesByRecordResponse, error) {
	return m.matches(r), nil
}
func (m *esaMock) GetCertificateWithOptions(r *esa.GetCertificateRequest, _ *dara.RuntimeOptions) (*esa.GetCertificateResponse, error) {
	return m.get(r), nil
}
func (m *esaMock) SetCertificateWithOptions(r *esa.SetCertificateRequest, _ *dara.RuntimeOptions) (*esa.SetCertificateResponse, error) {
	return m.set(r), nil
}
func esaSites() *esa.ListSitesResponse {
	return response[esa.ListSitesResponse](`{"StatusCode":200,"Body":{"TotalCount":1,"Sites":[{"SiteId":1,"SiteName":"example.com","AccessType":"NS"}]}}`)
}
func TestESADNSPaginationAndOwnedID(t *testing.T) {
	pages := 0
	m := &esaMock{list: func(r *esa.ListSitesRequest) *esa.ListSitesResponse {
		pages++
		if tea.Int32Value(r.PageNumber) != int32(pages) {
			t.Fatal("bad page")
		}
		if pages == 1 {
			return response[esa.ListSitesResponse](`{"StatusCode":200,"Body":{"TotalCount":2,"Sites":[{"SiteId":1,"SiteName":"example.com","AccessType":"NS"}]}}`)
		}
		return response[esa.ListSitesResponse](`{"StatusCode":200,"Body":{"TotalCount":2,"Sites":[{"SiteId":2,"SiteName":"sub.example.com","AccessType":"NS"}]}}`)
	}, create: func(r *esa.CreateRecordRequest) *esa.CreateRecordResponse {
		if tea.Int64Value(r.SiteId) != 2 || tea.StringValue(r.RecordName) != "_acme-challenge.sub.example.com" || tea.StringValue(r.Type) != "TXT" || tea.StringValue(r.Data.Value) != "fake-validation" {
			t.Fatal("incorrect TXT request")
		}
		return response[esa.CreateRecordResponse](`{"StatusCode":200,"Body":{"RecordId":101}}`)
	}, remove: func(r *esa.DeleteRecordRequest) *esa.DeleteRecordResponse {
		if tea.Int64Value(r.RecordId) != 101 {
			t.Fatal("wrong record deleted")
		}
		return response[esa.DeleteRecordResponse](`{"StatusCode":200,"Body":{}}`)
	}}
	e := &ESA{client: m}
	record, err := e.CreateTXT(context.Background(), "_acme-challenge.sub.example.com.", "fake-validation")
	if err != nil {
		t.Fatal(err)
	}
	if err = e.DeleteTXT(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	if pages != 2 {
		t.Fatal("cleanup queried sites")
	}
}
func TestESASearchAndUpdate(t *testing.T) {
	public, key, end := testCertificate(t, "www.example.com")
	m := &esaMock{list: func(*esa.ListSitesRequest) *esa.ListSitesResponse { return esaSites() }, matches: func(r *esa.ListCertificatesByRecordRequest) *esa.ListCertificatesByRecordResponse {
		if tea.StringValue(r.RecordName) != "www.example.com" || tea.BoolValue(r.ValidOnly) {
			t.Fatal("invalid matching query")
		}
		return response[esa.ListCertificatesByRecordResponse](`{"StatusCode":200,"Body":{"Result":[{"RecordName":"www.example.com","Count":1,"Certificates":[{"Id":"own","Name":"AutoCert-www.example.com","Type":"upload","Status":"OK","NotAfter":"wrong-unzoned-time"}]}]}}`)
	}, get: func(*esa.GetCertificateRequest) *esa.GetCertificateResponse {
		return &esa.GetCertificateResponse{StatusCode: tea.Int32(200), Body: &esa.GetCertificateResponseBody{Certificate: tea.String(public)}}
	}, set: func(r *esa.SetCertificateRequest) *esa.SetCertificateResponse {
		if tea.StringValue(r.Id) != "own" || tea.StringValue(r.Type) != "upload" || tea.StringValue(r.Certificate) != public || tea.StringValue(r.PrivateKey) != key {
			t.Fatal("certificate update request mismatch")
		}
		return response[esa.SetCertificateResponse](`{"StatusCode":200,"Body":{"Id":"own"}}`)
	}}
	e := &ESA{client: m}
	certs, err := e.Certificates(context.Background(), "www.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(certs) != 1 || !certs[0].Expires.Equal(end) {
		t.Fatal("expiry not read from PEM")
	}
	if err = e.Deploy(context.Background(), "www.example.com", public, key); err != nil {
		t.Fatal(err)
	}
}
func TestESAInvalidPages(t *testing.T) {
	for _, rsp := range []*esa.ListSitesResponse{nil, response[esa.ListSitesResponse](`{"StatusCode":500,"Body":{}}`), response[esa.ListSitesResponse](`{"StatusCode":200,"Body":{"TotalCount":1,"Sites":[]}}`)} {
		e := &ESA{client: &esaMock{list: func(*esa.ListSitesRequest) *esa.ListSitesResponse { return rsp }}}
		if _, err := e.site(context.Background(), "example.com", false); err == nil {
			t.Fatal("invalid page accepted")
		}
	}
}
func TestESAClientSTS(t *testing.T) {
	e, err := NewESA("fake-id", "fake-secret", "fake-token", "ap-southeast-1", "001")
	if err != nil {
		t.Fatal(err)
	}
	if e.SiteID != "1" {
		t.Fatal("numeric site ID was not canonicalized")
	}
	client := e.client.(*esa.Client)
	token, err := client.GetSecurityToken()
	if err != nil || tea.StringValue(token) != "fake-token" {
		t.Fatal("STS missing")
	}
	if tea.StringValue(client.Endpoint) != "esa.ap-southeast-1.aliyuncs.com" {
		t.Fatal("incorrect endpoint")
	}
	if _, err = NewESA("id", "key", "", "", "invalid"); err == nil {
		t.Fatal("invalid ID accepted")
	}
}

func TestESASANChangeCreatesDistinctCertificate(t *testing.T) {
	old, _, _ := testCertificate(t, "www.example.com", "extra.example.com")
	public, key, _ := testCertificate(t, "www.example.com")
	m := &esaMock{list: func(*esa.ListSitesRequest) *esa.ListSitesResponse { return esaSites() }, matches: func(*esa.ListCertificatesByRecordRequest) *esa.ListCertificatesByRecordResponse {
		return response[esa.ListCertificatesByRecordResponse](`{"StatusCode":200,"Body":{"Result":[{"RecordName":"www.example.com","Certificates":[{"Id":"shared","Name":"AutoCert-www.example.com","Type":"upload"}]}]}}`)
	}, get: func(*esa.GetCertificateRequest) *esa.GetCertificateResponse {
		return &esa.GetCertificateResponse{StatusCode: tea.Int32(200), Body: &esa.GetCertificateResponseBody{Certificate: tea.String(old)}}
	}, set: func(r *esa.SetCertificateRequest) *esa.SetCertificateResponse {
		if tea.StringValue(r.Id) != "" || tea.StringValue(r.Name) == "AutoCert-www.example.com" {
			t.Fatal("SAN change overwrote shared certificate or reused conflicting name")
		}
		return response[esa.SetCertificateResponse](`{"StatusCode":200,"Body":{"Id":"new"}}`)
	}}
	e := &ESA{client: m}
	if err := e.Deploy(context.Background(), "www.example.com", public, key); err != nil {
		t.Fatal(err)
	}
}
