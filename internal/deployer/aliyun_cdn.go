package deployer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cdn20180510 "github.com/alibabacloud-go/cdn-20180510/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/tiyee/AutoCert/internal/applicant"
)

func init() {
	RegisterDeployer("aliyun", NewAliyunCdn)
}

type aliyunAPI interface {
	SetCdnDomainSSLCertificateWithOptions(*cdn20180510.SetCdnDomainSSLCertificateRequest, *util.RuntimeOptions) (*cdn20180510.SetCdnDomainSSLCertificateResponse, error)
	DescribeDomainCertificateInfoWithOptions(*cdn20180510.DescribeDomainCertificateInfoRequest, *util.RuntimeOptions) (*cdn20180510.DescribeDomainCertificateInfoResponse, error)
}

type AliyunCdn struct {
	client  aliyunAPI
	options *Options
}

func NewAliyunCdn(option ...Option) (IDeployer, error) {
	opts := &Options{}
	opts.Update(option...)
	a := &AliyunCdn{
		options: opts,
	}
	client, err := a.createClient(opts.AccessKeyId, opts.AccessKeySecret, opts.SecurityToken)
	if err != nil {
		return nil, err
	}

	return &AliyunCdn{
		client:  client,
		options: opts,
	}, nil
}

func (a *AliyunCdn) Deploy(ctx context.Context) error {
	certName := fmt.Sprintf("%s-%s", a.options.Domain, time.Now().Format(time.DateTime))
	setCdnDomainSSLCertificateRequest := &cdn20180510.SetCdnDomainSSLCertificateRequest{
		DomainName:  tea.String(a.options.Domain),
		CertName:    tea.String(certName),
		CertType:    tea.String("upload"),
		SSLProtocol: tea.String("on"),
		SSLPub:      tea.String(a.options.Certificate.Certificate),
		SSLPri:      tea.String(a.options.Certificate.PrivateKey),
		CertRegion:  tea.String("cn-hangzhou"),
	}

	runtime := &util.RuntimeOptions{}

	resp, err := a.client.SetCdnDomainSSLCertificateWithOptions(setCdnDomainSSLCertificateRequest, runtime)
	if err != nil {
		return err
	}
	if resp == nil || resp.StatusCode == nil {
		return fmt.Errorf("deploy returned an empty response")
	}
	if tea.Int32Value(resp.StatusCode) < 200 || tea.Int32Value(resp.StatusCode) >= 300 {
		return fmt.Errorf("set cdn domain ssl certificate failed")
	}

	return nil
}

func (a *AliyunCdn) Search(domain string) ([]CertStat, error) {
	resp, err := a.client.DescribeDomainCertificateInfoWithOptions(&cdn20180510.DescribeDomainCertificateInfoRequest{DomainName: tea.String(domain)}, &util.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.StatusCode == nil || tea.Int32Value(resp.StatusCode) != http.StatusOK || resp.Body == nil {
		return nil, fmt.Errorf("query domain certificate returned an invalid response")
	}
	if resp.Body.CertInfos == nil {
		return nil, nil
	}
	var results []CertStat
	for _, item := range resp.Body.CertInfos.CertInfo {
		if item == nil {
			return nil, fmt.Errorf("query domain certificate returned an empty item")
		}
		if tea.StringValue(item.ServerCertificateStatus) != "on" {
			continue
		}
		end, err := time.Parse(time.RFC3339, tea.StringValue(item.CertExpireTime))
		if err != nil {
			return nil, fmt.Errorf("invalid domain certificate expiration: %w", err)
		}
		results = append(results, CertStat{LastTime: end.Unix(), CertName: tea.StringValue(item.CertName), CertId: tea.StringValue(item.CertId)})
	}
	return results, nil
}

func (a *AliyunCdn) SetCertificate(cert applicant.Certificate) {
	a.options.Certificate = cert
}

func (a *AliyunCdn) createClient(accessKeyId, accessKeySecret, securityToken string) (_result *cdn20180510.Client, _err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
		SecurityToken:   tea.String(securityToken),
	}
	config.Endpoint = tea.String("cdn.aliyuncs.com")
	_result = &cdn20180510.Client{}
	_result, _err = cdn20180510.NewClient(config)
	return _result, _err
}
