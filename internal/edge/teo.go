package edge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
	teo "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/teo/v20220901"
)

type teoAPI interface {
	DescribeZonesWithContext(context.Context, *teo.DescribeZonesRequest) (*teo.DescribeZonesResponse, error)
	CreateDnsRecordWithContext(context.Context, *teo.CreateDnsRecordRequest) (*teo.CreateDnsRecordResponse, error)
	DeleteDnsRecordsWithContext(context.Context, *teo.DeleteDnsRecordsRequest) (*teo.DeleteDnsRecordsResponse, error)
	DescribeAccelerationDomainsWithContext(context.Context, *teo.DescribeAccelerationDomainsRequest) (*teo.DescribeAccelerationDomainsResponse, error)
	ModifyHostsCertificateWithContext(context.Context, *teo.ModifyHostsCertificateRequest) (*teo.ModifyHostsCertificateResponse, error)
}
type sslAPI interface {
	UploadCertificateWithContext(context.Context, *ssl.UploadCertificateRequest) (*ssl.UploadCertificateResponse, error)
}
type TEO struct {
	client teoAPI
	ssl    sslAPI
	SiteID string
}

func NewTEO(id, secret, token, region, siteID string) (*TEO, error) {
	if siteID != "" && (!strings.HasPrefix(siteID, "zone-") || len(siteID) <= 5) {
		return nil, fmt.Errorf("TEO site ID must be a zone-* ID")
	}
	cred := common.NewTokenCredential(id, secret, token)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqTimeout = 100
	cpf.HttpProfile.Endpoint = "teo.tencentcloudapi.com"
	client, err := teo.NewClient(cred, region, cpf)
	if err != nil {
		return nil, apiError("TEO client", err)
	}
	sp := profile.NewClientProfile()
	sp.HttpProfile.ReqTimeout = 100
	sc, err := ssl.NewClient(cred, region, sp)
	if err != nil {
		return nil, apiError("SSL client", err)
	}
	return &TEO{client: client, ssl: sc, SiteID: siteID}, nil
}
func (t *TEO) site(ctx context.Context, domain string, dns bool) (Site, error) {
	var sites []Site
	for offset := int64(0); ; {
		req := teo.NewDescribeZonesRequest()
		req.Offset = tea.Int64(offset)
		req.Limit = tea.Int64(100)
		rsp, err := t.client.DescribeZonesWithContext(ctx, req)
		if err != nil {
			return Site{}, apiError("TEO DescribeZones", err)
		}
		if rsp == nil || rsp.Response == nil || rsp.Response.TotalCount == nil || *rsp.Response.TotalCount < 0 {
			return Site{}, fmt.Errorf("invalid TEO zone list")
		}
		for _, v := range rsp.Response.Zones {
			if v == nil {
				return Site{}, fmt.Errorf("empty TEO zone")
			}
			sites = append(sites, Site{ID: tea.StringValue(v.ZoneId), Name: tea.StringValue(v.ZoneName), DNS: tea.StringValue(v.Type) == "full"})
		}
		offset += int64(len(rsp.Response.Zones))
		if offset >= *rsp.Response.TotalCount {
			break
		}
		if len(rsp.Response.Zones) == 0 {
			return Site{}, fmt.Errorf("incomplete TEO zone list")
		}
	}
	return SelectSite(sites, domain, t.SiteID, dns)
}
func (t *TEO) CreateTXT(ctx context.Context, fqdn, value string) (Record, error) {
	fqdn, err := Normalize(fqdn)
	if err != nil {
		return Record{}, err
	}
	site, err := t.site(ctx, fqdn, true)
	if err != nil {
		return Record{}, err
	}
	req := teo.NewCreateDnsRecordRequest()
	req.ZoneId = tea.String(site.ID)
	req.Name = tea.String(fqdn)
	req.Type = tea.String("TXT")
	req.Content = tea.String(value)
	req.TTL = tea.Int64(60)
	rsp, err := t.client.CreateDnsRecordWithContext(ctx, req)
	if err != nil {
		return Record{}, apiError("TEO CreateDnsRecord", err)
	}
	if rsp == nil || rsp.Response == nil || tea.StringValue(rsp.Response.RecordId) == "" {
		return Record{}, fmt.Errorf("TEO returned no record ID")
	}
	return Record{SiteID: site.ID, ID: *rsp.Response.RecordId}, nil
}
func (t *TEO) DeleteTXT(ctx context.Context, record Record) error {
	if record.SiteID == "" || record.ID == "" {
		return fmt.Errorf("invalid TEO record handle")
	}
	req := teo.NewDeleteDnsRecordsRequest()
	req.ZoneId = tea.String(record.SiteID)
	req.RecordIds = []*string{tea.String(record.ID)}
	rsp, err := t.client.DeleteDnsRecordsWithContext(ctx, req)
	if err != nil {
		return apiError("TEO DeleteDnsRecords", err)
	}
	if rsp == nil || rsp.Response == nil {
		return fmt.Errorf("invalid TEO delete response")
	}
	return nil
}
func (t *TEO) host(ctx context.Context, site Site, domain string) (*teo.AccelerationDomain, error) {
	var found *teo.AccelerationDomain
	for offset := int64(0); ; {
		req := teo.NewDescribeAccelerationDomainsRequest()
		req.ZoneId = tea.String(site.ID)
		req.Offset = tea.Int64(offset)
		req.Limit = tea.Int64(100)
		rsp, err := t.client.DescribeAccelerationDomainsWithContext(ctx, req)
		if err != nil {
			return nil, apiError("TEO DescribeAccelerationDomains", err)
		}
		if rsp == nil || rsp.Response == nil || rsp.Response.TotalCount == nil || *rsp.Response.TotalCount < 0 {
			return nil, fmt.Errorf("invalid TEO host list")
		}
		for _, v := range rsp.Response.AccelerationDomains {
			if v == nil {
				return nil, fmt.Errorf("empty TEO host")
			}
			name, err := Normalize(tea.StringValue(v.DomainName))
			if err != nil {
				return nil, err
			}
			if name == domain {
				if found != nil {
					return nil, fmt.Errorf("duplicate TEO target domain")
				}
				found = v
			}
		}
		offset += int64(len(rsp.Response.AccelerationDomains))
		if offset >= *rsp.Response.TotalCount {
			break
		}
		if len(rsp.Response.AccelerationDomains) == 0 {
			return nil, fmt.Errorf("incomplete TEO host list")
		}
	}
	if found == nil {
		return nil, fmt.Errorf("TEO acceleration domain %s does not exist; create it before deployment", domain)
	}
	if tea.StringValue(found.IdentificationStatus) == "pending" {
		return nil, fmt.Errorf("TEO target domain ownership verification is pending")
	}
	if status := tea.StringValue(found.DomainStatus); status == "offline" || status == "init" {
		return nil, fmt.Errorf("TEO target domain is not active")
	}
	return found, nil
}
func (t *TEO) Certificates(ctx context.Context, domain string) ([]Certificate, error) {
	domain, err := Normalize(domain)
	if err != nil {
		return nil, err
	}
	site, err := t.site(ctx, domain, false)
	if err != nil {
		return nil, err
	}
	host, err := t.host(ctx, site, domain)
	if err != nil {
		return nil, err
	}
	if host.Certificate == nil || tea.StringValue(host.Certificate.Mode) == "disable" {
		return nil, nil
	}
	var certs []Certificate
	for _, v := range host.Certificate.List {
		if v == nil {
			return nil, fmt.Errorf("empty TEO certificate")
		}
		status := tea.StringValue(v.Status)
		if status == "processing" || status == "applying" {
			return nil, fmt.Errorf("TEO certificate deployment is still in progress")
		}
		if status != "deployed" {
			continue
		}
		end, err := time.Parse(time.RFC3339, tea.StringValue(v.ExpireTime))
		if err != nil {
			return nil, fmt.Errorf("invalid TEO certificate expiry")
		}
		if tea.StringValue(v.CertId) == "" {
			return nil, fmt.Errorf("TEO certificate has no ID")
		}
		certs = append(certs, Certificate{ID: *v.CertId, Name: tea.StringValue(v.Alias), Expires: end})
	}
	return certs, nil
}
func (t *TEO) Deploy(ctx context.Context, domain, public, private string) error {
	domain, err := Normalize(domain)
	if err != nil {
		return err
	}
	cert, err := validateDeployment(public, private, domain)
	if err != nil {
		return err
	}
	site, err := t.site(ctx, domain, false)
	if err != nil {
		return err
	}
	if _, err = t.host(ctx, site, domain); err != nil {
		return err
	}
	req := ssl.NewUploadCertificateRequest()
	req.CertificatePublicKey = tea.String(public)
	req.CertificatePrivateKey = tea.String(private)
	req.Alias = tea.String(certificateName(domain, cert.DNSNames))
	upload, err := t.ssl.UploadCertificateWithContext(ctx, req)
	if err != nil {
		return apiError("SSL UploadCertificate", err)
	}
	if upload == nil || upload.Response == nil || tea.StringValue(upload.Response.CertificateId) == "" {
		return fmt.Errorf("SSL returned no certificate ID")
	}
	bind := teo.NewModifyHostsCertificateRequest()
	bind.ZoneId = tea.String(site.ID)
	bind.Hosts = []*string{tea.String(domain)}
	bind.Mode = tea.String("sslcert")
	bind.ServerCertInfo = []*teo.ServerCertInfo{{CertId: upload.Response.CertificateId}}
	rsp, err := t.client.ModifyHostsCertificateWithContext(ctx, bind)
	if err != nil {
		return fmt.Errorf("SSL certificate %s uploaded, but binding failed: %w", *upload.Response.CertificateId, apiError("TEO ModifyHostsCertificate", err))
	}
	if rsp == nil || rsp.Response == nil {
		return fmt.Errorf("invalid TEO deploy response")
	}
	return nil
}

func (t *TEO) CheckTarget(ctx context.Context, domain string) error {
	domain, err := Normalize(domain)
	if err != nil {
		return err
	}
	site, err := t.site(ctx, domain, false)
	if err != nil {
		return err
	}
	_, err = t.host(ctx, site, domain)
	return err
}
