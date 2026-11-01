package edge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
	teo "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/teo/v20220901"
)

type teoMock struct {
	teoAPI
	zones  func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse
	create func(*teo.CreateDnsRecordRequest) *teo.CreateDnsRecordResponse
	remove func(*teo.DeleteDnsRecordsRequest) *teo.DeleteDnsRecordsResponse
	hosts  func(*teo.DescribeAccelerationDomainsRequest) *teo.DescribeAccelerationDomainsResponse
	bind   func(*teo.ModifyHostsCertificateRequest) (*teo.ModifyHostsCertificateResponse, error)
}

func (m *teoMock) DescribeZonesWithContext(_ context.Context, r *teo.DescribeZonesRequest) (*teo.DescribeZonesResponse, error) {
	return m.zones(r), nil
}
func (m *teoMock) CreateDnsRecordWithContext(_ context.Context, r *teo.CreateDnsRecordRequest) (*teo.CreateDnsRecordResponse, error) {
	return m.create(r), nil
}
func (m *teoMock) DeleteDnsRecordsWithContext(_ context.Context, r *teo.DeleteDnsRecordsRequest) (*teo.DeleteDnsRecordsResponse, error) {
	return m.remove(r), nil
}
func (m *teoMock) DescribeAccelerationDomainsWithContext(_ context.Context, r *teo.DescribeAccelerationDomainsRequest) (*teo.DescribeAccelerationDomainsResponse, error) {
	return m.hosts(r), nil
}
func (m *teoMock) ModifyHostsCertificateWithContext(_ context.Context, r *teo.ModifyHostsCertificateRequest) (*teo.ModifyHostsCertificateResponse, error) {
	return m.bind(r)
}

type sslMock struct {
	calls   int
	err     error
	request *ssl.UploadCertificateRequest
}

func (m *sslMock) UploadCertificateWithContext(_ context.Context, r *ssl.UploadCertificateRequest) (*ssl.UploadCertificateResponse, error) {
	m.calls++
	m.request = r
	return response[ssl.UploadCertificateResponse](`{"Response":{"CertificateId":"uploaded"}}`), m.err
}
func teoZones() *teo.DescribeZonesResponse {
	return response[teo.DescribeZonesResponse](`{"Response":{"TotalCount":1,"Zones":[{"ZoneId":"zone-one","ZoneName":"example.com","Type":"full"}]}}`)
}
func teoHost() *teo.DescribeAccelerationDomainsResponse {
	return response[teo.DescribeAccelerationDomainsResponse](`{"Response":{"TotalCount":1,"AccelerationDomains":[{"DomainName":"www.example.com","Certificate":{"Mode":"sslcert","List":[{"CertId":"bound","Status":"deployed","ExpireTime":"2027-01-01T00:00:00Z"}]}}]}}`)
}
func TestTEODNSOwnedRecord(t *testing.T) {
	m := &teoMock{zones: func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse { return teoZones() }, create: func(r *teo.CreateDnsRecordRequest) *teo.CreateDnsRecordResponse {
		if tea.StringValue(r.ZoneId) != "zone-one" || tea.StringValue(r.Name) != "_acme-challenge.example.com" || tea.StringValue(r.Type) != "TXT" || tea.StringValue(r.Content) != "fake-value" {
			t.Fatal("invalid TXT request")
		}
		return response[teo.CreateDnsRecordResponse](`{"Response":{"RecordId":"record-own"}}`)
	}, remove: func(r *teo.DeleteDnsRecordsRequest) *teo.DeleteDnsRecordsResponse {
		if tea.StringValue(r.ZoneId) != "zone-one" || len(r.RecordIds) != 1 || *r.RecordIds[0] != "record-own" {
			t.Fatal("deleting unowned record")
		}
		return response[teo.DeleteDnsRecordsResponse](`{"Response":{}}`)
	}}
	c := &TEO{client: m}
	r, err := c.CreateTXT(context.Background(), "_acme-challenge.example.com.", "fake-value")
	if err != nil {
		t.Fatal(err)
	}
	m.zones = func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse {
		t.Fatal("cleanup re-discovered zone")
		return nil
	}
	if err = c.DeleteTXT(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}
func TestTEOUploadAndBind(t *testing.T) {
	public, key, _ := testCertificate(t, "www.example.com")
	upload := &sslMock{}
	m := &teoMock{zones: func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse { return teoZones() }, hosts: func(*teo.DescribeAccelerationDomainsRequest) *teo.DescribeAccelerationDomainsResponse {
		return teoHost()
	}, bind: func(r *teo.ModifyHostsCertificateRequest) (*teo.ModifyHostsCertificateResponse, error) {
		if upload.calls != 1 || tea.StringValue(r.ZoneId) != "zone-one" || len(r.Hosts) != 1 || *r.Hosts[0] != "www.example.com" || tea.StringValue(r.Mode) != "sslcert" || len(r.ServerCertInfo) != 1 || *r.ServerCertInfo[0].CertId != "uploaded" {
			t.Fatal("not binding uploaded certificate to exact target")
		}
		if r.ClientCertInfo != nil || r.UpstreamCertInfo != nil {
			t.Fatal("unrelated TLS settings modified")
		}
		return response[teo.ModifyHostsCertificateResponse](`{"Response":{}}`), nil
	}}
	c := &TEO{client: m, ssl: upload}
	certs, err := c.Certificates(context.Background(), "www.example.com")
	if err != nil || len(certs) != 1 || certs[0].ID != "bound" {
		t.Fatalf("%v %v", certs, err)
	}
	if err = c.Deploy(context.Background(), "www.example.com", public, key); err != nil {
		t.Fatal(err)
	}
	if tea.StringValue(upload.request.CertificatePublicKey) != public || tea.StringValue(upload.request.CertificatePrivateKey) != key {
		t.Fatal("wrong certificate uploaded")
	}
	m.bind = func(*teo.ModifyHostsCertificateRequest) (*teo.ModifyHostsCertificateResponse, error) {
		return nil, errors.New("private-content")
	}
	if err = c.Deploy(context.Background(), "www.example.com", public, key); err == nil || strings.Contains(err.Error(), "private-content") {
		t.Fatal("binding error lost or leaked")
	}
	upload.err = errors.New("upload error")
	m.bind = func(*teo.ModifyHostsCertificateRequest) (*teo.ModifyHostsCertificateResponse, error) {
		t.Fatal("bound after failed upload")
		return nil, nil
	}
	if err = c.Deploy(context.Background(), "www.example.com", public, key); err == nil {
		t.Fatal("upload failure lost")
	}
}
func TestTEOHostPaginationAndPendingCertificate(t *testing.T) {
	calls := 0
	m := &teoMock{zones: func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse { return teoZones() }, hosts: func(r *teo.DescribeAccelerationDomainsRequest) *teo.DescribeAccelerationDomainsResponse {
		calls++
		if tea.Int64Value(r.Offset) == 0 {
			return response[teo.DescribeAccelerationDomainsResponse](`{"Response":{"TotalCount":2,"AccelerationDomains":[{"DomainName":"other.example.com"}]}}`)
		}
		rsp := teoHost()
		rsp.Response.TotalCount = tea.Int64(2)
		return rsp
	}}
	c := &TEO{client: m}
	certs, err := c.Certificates(context.Background(), "www.example.com")
	if err != nil || len(certs) != 1 || calls != 2 {
		t.Fatal("host pagination failed", err)
	}
	m.hosts = func(*teo.DescribeAccelerationDomainsRequest) *teo.DescribeAccelerationDomainsResponse {
		rsp := teoHost()
		rsp.Response.AccelerationDomains[0].Certificate.List[0].Status = tea.String("processing")
		return rsp
	}
	if _, err = c.Certificates(context.Background(), "www.example.com"); err == nil {
		t.Fatal("pending deployment treated as missing")
	}
	m.zones = func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse {
		return response[teo.DescribeZonesResponse](`{"Response":{"TotalCount":1,"Zones":[]}}`)
	}
	if _, err = c.site(context.Background(), "example.com", false); err == nil {
		t.Fatal("incomplete list accepted")
	}
}
func TestTEOClientSTS(t *testing.T) {
	c, err := NewTEO("fake-id", "fake-secret", "fake-token", "", "zone-one")
	if err != nil {
		t.Fatal(err)
	}
	if c.client.(*teo.Client).GetCredential().GetToken() != "fake-token" || c.ssl.(*ssl.Client).GetCredential().GetToken() != "fake-token" {
		t.Fatal("STS not passed to both clients")
	}
	if _, err = NewTEO("id", "key", "", "", "1"); err == nil {
		t.Fatal("invalid zone ID accepted")
	}
}

func TestTEOPreflightRejectsUnusableTarget(t *testing.T) {
	public, key, _ := testCertificate(t, "www.example.com")
	upload := &sslMock{}
	m := &teoMock{zones: func(*teo.DescribeZonesRequest) *teo.DescribeZonesResponse { return teoZones() }}
	c := &TEO{client: m, ssl: upload}
	for _, state := range []string{"missing", "pending", "offline"} {
		m.hosts = func(*teo.DescribeAccelerationDomainsRequest) *teo.DescribeAccelerationDomainsResponse {
			rsp := teoHost()
			host := rsp.Response.AccelerationDomains[0]
			switch state {
			case "missing":
				host.DomainName = tea.String("other.example.com")
			case "pending":
				host.IdentificationStatus = tea.String("pending")
			case "offline":
				host.DomainStatus = tea.String("offline")
			}
			return rsp
		}
		if err := c.CheckTarget(context.Background(), "www.example.com"); err == nil {
			t.Fatalf("%s preflight accepted", state)
		}
		if err := c.Deploy(context.Background(), "www.example.com", public, key); err == nil {
			t.Fatalf("%s target deployed", state)
		}
	}
	if upload.calls != 0 {
		t.Fatal("uploaded despite unusable target")
	}
}
