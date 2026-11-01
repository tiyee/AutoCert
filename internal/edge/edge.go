package edge

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	tcerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"golang.org/x/net/idna"
)

type Site struct {
	ID, Name string
	DNS      bool
}
type Certificate struct {
	ID, Name, Fingerprint string
	Expires               time.Time
}
type Record struct{ SiteID, ID string }

func Normalize(name string) (string, error) {
	name = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	if name == "" {
		return "", fmt.Errorf("empty domain")
	}
	wildcard := ""
	if strings.HasPrefix(name, "*.") {
		wildcard = "*."
		name = strings.TrimPrefix(name, "*.")
	}
	ascii, err := idna.ToASCII(name)
	if err != nil {
		return "", fmt.Errorf("invalid domain")
	}
	if strings.Contains(ascii, "*") {
		return "", fmt.Errorf("invalid wildcard domain")
	}
	if len(ascii) > 253 {
		return "", fmt.Errorf("invalid domain length")
	}
	for _, label := range strings.Split(ascii, ".") {
		if len(label) == 0 || len(label) > 63 {
			return "", fmt.Errorf("invalid DNS label")
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
				return "", fmt.Errorf("invalid DNS label")
			}
		}
	}
	return wildcard + ascii, nil
}
func SelectSite(sites []Site, domain, id string, dns bool) (Site, error) {
	domain, err := Normalize(domain)
	if err != nil {
		return Site{}, err
	}
	domain = strings.TrimPrefix(domain, "*.")
	var best Site
	ambiguous := false
	for _, s := range sites {
		if id != "" && s.ID != id {
			continue
		}
		// TEO 无域名接入站点没有可匹配的 DNS 名称。
		if strings.TrimSpace(s.Name) == "" {
			if id != "" {
				return Site{}, fmt.Errorf("selected site has no domain")
			}
			continue
		}
		name, err := Normalize(s.Name)
		if err != nil {
			if id != "" {
				return Site{}, err
			}
			continue
		}
		if domain != name && !strings.HasSuffix(domain, "."+name) {
			continue
		}
		if s.ID == "" {
			return Site{}, fmt.Errorf("site has no ID")
		}
		if len(name) > len(best.Name) {
			s.Name = name
			best = s
			ambiguous = false
		} else if len(name) == len(best.Name) && s.ID != best.ID {
			ambiguous = true
		}
	}
	if ambiguous {
		return Site{}, fmt.Errorf("ambiguous site; specify dns-site-id or cdn-site-id")
	}
	if best.ID == "" {
		return Site{}, fmt.Errorf("no matching site for %s", domain)
	}
	if dns && !best.DNS {
		return Site{}, fmt.Errorf("site %s is not NS hosted; select its authoritative DNS provider", best.ID)
	}
	return best, nil
}
func ParseCertificate(value, domain string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid X509 certificate")
	}
	domain, err = Normalize(domain)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(domain, "*.") {
		for _, name := range cert.DNSNames {
			if strings.EqualFold(name, domain) {
				return cert, nil
			}
		}
		return nil, fmt.Errorf("certificate does not cover requested wildcard")
	}
	if cert.VerifyHostname(domain) != nil {
		return nil, fmt.Errorf("certificate does not cover requested domain")
	}
	return cert, nil
}

// API 错误可能包含请求正文（含私钥），只保留错误码。
func apiError(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	code := "transport"
	var modern interface{ GetCode() *string }
	var ali *tea.SDKError
	var tc *tcerrors.TencentCloudSDKError
	if errors.As(err, &modern) {
		code = tea.StringValue(modern.GetCode())
	} else if errors.As(err, &ali) {
		code = tea.StringValue(ali.Code)
	} else if errors.As(err, &tc) {
		code = tc.Code
	}
	return fmt.Errorf("%s failed (%s)", action, code)
}

func validateDeployment(public, private, domain string) (*x509.Certificate, error) {
	cert, err := ParseCertificate(public, domain)
	if err != nil {
		return nil, err
	}
	if _, err := tls.X509KeyPair([]byte(public), []byte(private)); err != nil {
		return nil, fmt.Errorf("certificate and private key do not match")
	}
	now := time.Now()
	if cert.NotBefore.After(now) {
		return nil, fmt.Errorf("cannot deploy a certificate that is not valid yet")
	}
	if !cert.NotAfter.After(now) {
		return nil, fmt.Errorf("cannot deploy an expired certificate")
	}
	return cert, nil
}

// 名称绑定域名和 SAN 范围，避免扩大/缩小申请范围时撞名；保持在 ESA 的 128 字节限制内。
func certificateName(domain string, names []string) string {
	names = canonicalNames(names)
	sum := sha256.Sum256([]byte(domain + "\x00" + strings.Join(names, "\x00")))
	if len(domain) > 88 {
		domain = domain[:88]
	}
	return fmt.Sprintf("AutoCert-%s-%x", domain, sum[:12])
}

func canonicalNames(names []string) []string {
	names = slices.Clone(names)
	for i, n := range names {
		names[i] = strings.ToLower(strings.TrimSuffix(n, "."))
	}
	slices.Sort(names)
	return slices.Compact(names)
}
