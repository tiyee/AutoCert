package edge

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	esa "github.com/alibabacloud-go/esa-20240910/v3/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type esaAPI interface {
	ListSitesWithOptions(*esa.ListSitesRequest, *dara.RuntimeOptions) (*esa.ListSitesResponse, error)
	CreateRecordWithOptions(*esa.CreateRecordRequest, *dara.RuntimeOptions) (*esa.CreateRecordResponse, error)
	DeleteRecordWithOptions(*esa.DeleteRecordRequest, *dara.RuntimeOptions) (*esa.DeleteRecordResponse, error)
	SetCertificateWithOptions(*esa.SetCertificateRequest, *dara.RuntimeOptions) (*esa.SetCertificateResponse, error)
	ListCertificatesByRecordWithOptions(*esa.ListCertificatesByRecordRequest, *dara.RuntimeOptions) (*esa.ListCertificatesByRecordResponse, error)
	GetCertificateWithOptions(*esa.GetCertificateRequest, *dara.RuntimeOptions) (*esa.GetCertificateResponse, error)
}
type ESA struct {
	client esaAPI
	SiteID string
}

func NewESA(id, secret, token, region, siteID string) (*ESA, error) {
	if region == "" {
		region = "cn-hangzhou"
	}
	if region != "cn-hangzhou" && region != "ap-southeast-1" {
		return nil, fmt.Errorf("unsupported ESA API region")
	}
	if siteID != "" {
		n, err := strconv.ParseInt(siteID, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("ESA site ID must be a positive integer")
		}
		siteID = strconv.FormatInt(n, 10)
	}

	client, err := esa.NewClient(&openapi.Config{AccessKeyId: tea.String(id), AccessKeySecret: tea.String(secret), SecurityToken: tea.String(token), RegionId: tea.String(region), Endpoint: tea.String("esa." + region + ".aliyuncs.com"), RetryOptions: &dara.RetryOptions{Retryable: false, MaxAttempts: 1}, ConnectTimeout: tea.Int(10000), ReadTimeout: tea.Int(100000)})
	if err != nil {
		return nil, apiError("ESA client", err)
	}
	return &ESA{client: client, SiteID: siteID}, nil
}
func esaRuntime() *dara.RuntimeOptions     { return &dara.RuntimeOptions{Autoretry: tea.Bool(false)} }
func validAli(code *int32, body bool) bool { return code != nil && *code >= 200 && *code < 300 && body }
func (e *ESA) site(ctx context.Context, domain string, dns bool) (Site, error) {
	var sites []Site
	for page := int32(1); ; page++ {
		if err := ctx.Err(); err != nil {
			return Site{}, err
		}
		rsp, err := e.client.ListSitesWithOptions(&esa.ListSitesRequest{PageNumber: tea.Int32(page), PageSize: tea.Int32(100)}, esaRuntime())
		if err != nil {
			return Site{}, apiError("ESA ListSites", err)
		}
		if rsp == nil || !validAli(rsp.StatusCode, rsp.Body != nil) || rsp.Body.TotalCount == nil || *rsp.Body.TotalCount < 0 {
			return Site{}, fmt.Errorf("invalid ESA site list")
		}
		for _, v := range rsp.Body.Sites {
			if v == nil || tea.Int64Value(v.SiteId) <= 0 {
				return Site{}, fmt.Errorf("empty ESA site")
			}
			sites = append(sites, Site{ID: strconv.FormatInt(tea.Int64Value(v.SiteId), 10), Name: tea.StringValue(v.SiteName), DNS: tea.StringValue(v.AccessType) == "NS"})
		}
		if len(sites) >= int(*rsp.Body.TotalCount) {
			break
		}
		if len(rsp.Body.Sites) == 0 {
			return Site{}, fmt.Errorf("incomplete ESA site list")
		}
	}
	return SelectSite(sites, domain, e.SiteID, dns)
}
func (e *ESA) CreateTXT(ctx context.Context, fqdn, value string) (Record, error) {
	fqdn, err := Normalize(fqdn)
	if err != nil {
		return Record{}, err
	}
	site, err := e.site(ctx, fqdn, true)
	if err != nil {
		return Record{}, err
	}
	id, _ := strconv.ParseInt(site.ID, 10, 64)
	rsp, err := e.client.CreateRecordWithOptions(&esa.CreateRecordRequest{SiteId: tea.Int64(id), RecordName: tea.String(fqdn), Type: tea.String("TXT"), Ttl: tea.Int32(60), Data: &esa.CreateRecordRequestData{Value: tea.String(value)}}, esaRuntime())
	if err != nil {
		return Record{}, apiError("ESA CreateRecord", err)
	}
	if rsp == nil || !validAli(rsp.StatusCode, rsp.Body != nil) || tea.Int64Value(rsp.Body.RecordId) <= 0 {
		return Record{}, fmt.Errorf("ESA returned no record ID")
	}
	return Record{SiteID: site.ID, ID: strconv.FormatInt(*rsp.Body.RecordId, 10)}, nil
}
func (e *ESA) DeleteTXT(ctx context.Context, record Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id, err := strconv.ParseInt(record.ID, 10, 64)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid ESA record ID")
	}
	rsp, err := e.client.DeleteRecordWithOptions(&esa.DeleteRecordRequest{RecordId: tea.Int64(id)}, esaRuntime())
	if err != nil {
		return apiError("ESA DeleteRecord", err)
	}
	if rsp == nil || !validAli(rsp.StatusCode, rsp.Body != nil) {
		return fmt.Errorf("invalid ESA delete response")
	}
	return nil
}
func (e *ESA) matching(site Site, domain string) ([]*esa.ListCertificatesByRecordResponseBodyResultCertificates, error) {
	id, _ := strconv.ParseInt(site.ID, 10, 64)
	rsp, err := e.client.ListCertificatesByRecordWithOptions(&esa.ListCertificatesByRecordRequest{SiteId: tea.Int64(id), RecordName: tea.String(domain), Detail: tea.Bool(true), ValidOnly: tea.Bool(false)}, esaRuntime())
	if err != nil {
		return nil, apiError("ESA ListCertificatesByRecord", err)
	}
	if rsp == nil || !validAli(rsp.StatusCode, rsp.Body != nil) {
		return nil, fmt.Errorf("invalid ESA certificate list")
	}
	var certs []*esa.ListCertificatesByRecordResponseBodyResultCertificates
	for _, v := range rsp.Body.Result {
		if v == nil {
			return nil, fmt.Errorf("empty ESA record result")
		}
		name, err := Normalize(tea.StringValue(v.RecordName))
		if err != nil {
			return nil, err
		}
		if name == domain {
			certs = append(certs, v.Certificates...)
		}
	}
	return certs, nil
}
func (e *ESA) certificate(site Site, id string, domain string) (string, error) {
	n, _ := strconv.ParseInt(site.ID, 10, 64)
	rsp, err := e.client.GetCertificateWithOptions(&esa.GetCertificateRequest{SiteId: tea.Int64(n), Id: tea.String(id)}, esaRuntime())
	if err != nil {
		return "", apiError("ESA GetCertificate", err)
	}
	if rsp == nil || !validAli(rsp.StatusCode, rsp.Body != nil) {
		return "", fmt.Errorf("invalid ESA certificate detail")
	}
	return tea.StringValue(rsp.Body.Certificate), nil
}
func (e *ESA) Certificates(ctx context.Context, domain string) ([]Certificate, error) {
	domain, err := Normalize(domain)
	if err != nil {
		return nil, err
	}
	site, err := e.site(ctx, domain, false)
	if err != nil {
		return nil, err
	}
	matches, err := e.matching(site, domain)
	if err != nil {
		return nil, err
	}
	var results []Certificate
	for _, v := range matches {
		if v == nil || tea.StringValue(v.Id) == "" {
			return nil, fmt.Errorf("ESA certificate has no ID")
		}
		if strings.EqualFold(tea.StringValue(v.Status), "revoked") {
			continue
		}
		content, err := e.certificate(site, *v.Id, domain)
		if err != nil {
			return nil, err
		}
		cert, err := ParseCertificate(content, domain)
		if err != nil {
			return nil, err
		}
		results = append(results, Certificate{ID: *v.Id, Name: tea.StringValue(v.Name), Fingerprint: tea.StringValue(v.FingerprintSha256), Expires: cert.NotAfter})
	}
	return results, nil
}
func (e *ESA) Deploy(ctx context.Context, domain, public, private string) error {
	domain, err := Normalize(domain)
	if err != nil {
		return err
	}
	cert, err := validateDeployment(public, private, domain)
	if err != nil {
		return err
	}
	site, err := e.site(ctx, domain, false)
	if err != nil {
		return err
	}
	matches, err := e.matching(site, domain)
	if err != nil {
		return err
	}
	name := certificateName(domain, cert.DNSNames)
	legacyName := "AutoCert-" + domain
	candidateName := name
	var updateID string
	for _, v := range matches {
		if v == nil || (tea.StringValue(v.Name) != candidateName && tea.StringValue(v.Name) != legacyName) || tea.StringValue(v.Type) != "upload" {
			continue
		}
		content, err := e.certificate(site, tea.StringValue(v.Id), domain)
		if err != nil {
			return err
		}
		old, err := ParseCertificate(content, domain)
		if err != nil {
			return err
		}
		oldNames, newNames := canonicalNames(old.DNSNames), canonicalNames(cert.DNSNames)
		if slices.Equal(oldNames, newNames) {
			if updateID != "" {
				return fmt.Errorf("multiple AutoCert ESA certificates match; resolve duplicates before updating")
			}
			updateID = tea.StringValue(v.Id)
			name = tea.StringValue(v.Name)
		}
	}
	id, _ := strconv.ParseInt(site.ID, 10, 64)
	req := &esa.SetCertificateRequest{SiteId: tea.Int64(id), Name: tea.String(name), Type: tea.String("upload"), Certificate: tea.String(public), PrivateKey: tea.String(private)}
	if updateID != "" {
		req.Id = tea.String(updateID)
	}
	rsp, err := e.client.SetCertificateWithOptions(req, esaRuntime())
	if err != nil {
		return apiError("ESA SetCertificate", err)
	}
	if rsp == nil || !validAli(rsp.StatusCode, rsp.Body != nil) || tea.StringValue(rsp.Body.Id) == "" {
		return fmt.Errorf("ESA returned no deployed certificate ID")
	}
	return nil
}

func (e *ESA) CheckTarget(ctx context.Context, domain string) error {
	domain, err := Normalize(domain)
	if err != nil {
		return err
	}
	site, err := e.site(ctx, domain, false)
	if err != nil {
		return err
	}
	_, err = e.matching(site, domain)
	return err
}
