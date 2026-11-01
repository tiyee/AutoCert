package deployer

import (
	"context"
	"errors"
	"testing"
	"time"

	cdn "github.com/alibabacloud-go/cdn-20180510/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
)

type fakeTencentAPI struct {
	pages []*ssl.DescribeCertificatesResponse
	calls int
}

func (f *fakeTencentAPI) UploadCertificateWithContext(context.Context, *ssl.UploadCertificateRequest) (*ssl.UploadCertificateResponse, error) {
	return nil, errors.New("unused")
}
func (f *fakeTencentAPI) DescribeCertificates(req *ssl.DescribeCertificatesRequest) (*ssl.DescribeCertificatesResponse, error) {
	if tea.Uint64Value(req.Offset) != uint64(f.calls) {
		return nil, errors.New("wrong offset")
	}
	rsp := f.pages[f.calls]
	f.calls++
	return rsp, nil
}
func tencentPage(total uint64, certs ...*ssl.Certificates) *ssl.DescribeCertificatesResponse {
	return &ssl.DescribeCertificatesResponse{Response: &ssl.DescribeCertificatesResponseParams{TotalCount: tea.Uint64(total), Certificates: certs}}
}
func TestTencentSearchPaginationAndFiltering(t *testing.T) {
	f := &fakeTencentAPI{pages: []*ssl.DescribeCertificatesResponse{
		tencentPage(3, &ssl.Certificates{Domain: tea.String("notexample.com"), Status: tea.Uint64(1)}),
		tencentPage(3, &ssl.Certificates{Domain: tea.String("*.example.com"), Status: tea.Uint64(1), CertEndTime: tea.String("2026-11-01 08:00:00")}),
		tencentPage(3, &ssl.Certificates{Domain: tea.String("www.example.com"), Status: tea.Uint64(10)}),
	}}
	d := &Tencent{client: f}
	certs, err := d.Search("www.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(certs) != 1 || f.calls != 3 {
		t.Fatalf("certs %d calls %d", len(certs), f.calls)
	}
	if time.Unix(certs[0].LastTime, 0).UTC().Format(time.RFC3339) != "2026-11-01T00:00:00Z" {
		t.Fatal(certs)
	}
}
func TestTencentSearchInvalidResponses(t *testing.T) {
	for _, rsp := range []*ssl.DescribeCertificatesResponse{nil, {Response: &ssl.DescribeCertificatesResponseParams{}}, tencentPage(1), tencentPage(1, &ssl.Certificates{Domain: tea.String("example.com"), Status: tea.Uint64(1)})} {
		d := &Tencent{client: &fakeTencentAPI{pages: []*ssl.DescribeCertificatesResponse{rsp}}}
		if _, err := d.Search("example.com"); err == nil {
			t.Fatal("invalid response accepted")
		}
	}
}

type fakeAliyunAPI struct {
	response *cdn.DescribeDomainCertificateInfoResponse
	domain   string
}

func (f *fakeAliyunAPI) SetCdnDomainSSLCertificateWithOptions(*cdn.SetCdnDomainSSLCertificateRequest, *util.RuntimeOptions) (*cdn.SetCdnDomainSSLCertificateResponse, error) {
	return nil, errors.New("unused")
}
func (f *fakeAliyunAPI) DescribeDomainCertificateInfoWithOptions(req *cdn.DescribeDomainCertificateInfoRequest, _ *util.RuntimeOptions) (*cdn.DescribeDomainCertificateInfoResponse, error) {
	f.domain = tea.StringValue(req.DomainName)
	return f.response, nil
}
func TestAliyunUsesBoundCertificateExpiry(t *testing.T) {
	f := &fakeAliyunAPI{response: &cdn.DescribeDomainCertificateInfoResponse{StatusCode: tea.Int32(200), Body: &cdn.DescribeDomainCertificateInfoResponseBody{CertInfos: &cdn.DescribeDomainCertificateInfoResponseBodyCertInfos{CertInfo: []*cdn.DescribeDomainCertificateInfoResponseBodyCertInfosCertInfo{{ServerCertificateStatus: tea.String("on"), CertExpireTime: tea.String("2026-11-01T00:00:00Z"), CertId: tea.String("bound")}}}}}}
	d := &AliyunCdn{client: f}
	certs, err := d.Search("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if f.domain != "example.com" || len(certs) != 1 || certs[0].CertId != "bound" {
		t.Fatal(certs)
	}
	if time.Unix(certs[0].LastTime, 0).UTC().Format(time.RFC3339) != "2026-11-01T00:00:00Z" {
		t.Fatal(certs)
	}
	f.response.StatusCode = tea.Int32(500)
	if _, err := d.Search("example.com"); err == nil {
		t.Fatal("HTTP error swallowed")
	}
	f.response.StatusCode = tea.Int32(200)
	f.response.Body.CertInfos.CertInfo[0].CertExpireTime = nil
	if _, err := d.Search("example.com"); err == nil {
		t.Fatal("empty expiry accepted")
	}
}
