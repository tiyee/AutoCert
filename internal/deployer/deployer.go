package deployer

import (
	"context"
	"time"

	"github.com/tiyee/AutoCert/internal/applicant"
)

type CertStat struct {
	LastTime    int64  `json:"last_time"`
	Fingerprint string `json:"fingerprint"`
	CertName    string `json:"cert_name"`
	CertId      string `json:"cert_id"`
}

type IDeployer interface {
	Deploy(ctx context.Context) error
	Search(domain string) ([]CertStat, error)
	SetCertificate(cert applicant.Certificate)
}

type deployerFactory = func(...Option) (IDeployer, error)

var applicantMap = map[string]deployerFactory{}

func GetDeployer(platform string) deployerFactory {
	return applicantMap[platform]
}

func RegisterDeployer(platform string, f deployerFactory) {
	applicantMap[platform] = f
}

func (s CertStat) Expired() bool {
	return s.LastTime <= time.Now().Unix()
}
func (s CertStat) Renewable() bool {
	return s.renewableAt(time.Now())
}

type CertStats []CertStat

func (c CertStats) Len() int {
	return len(c)
}

func (c CertStats) Less(i, j int) bool {
	return c[i].LastTime < c[j].LastTime
}

func (c CertStats) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (s CertStat) renewableAt(now time.Time) bool {
	return s.LastTime <= now.Add(10*24*time.Hour).Unix()
}
func LatestCertificate(certs []CertStat) (CertStat, bool) {
	if len(certs) == 0 {
		return CertStat{}, false
	}
	latest := certs[0]
	for _, cert := range certs[1:] {
		if cert.LastTime > latest.LastTime {
			latest = cert
		}
	}
	return latest, true
}

// PreflightDeployer 在签发前执行只读目标校验；现有第三方适配器可选实现。
type PreflightDeployer interface{ Preflight(context.Context) error }

// RenewalSelector 区分证书库候选证书与线上同时绑定的证书。
type RenewalSelector interface {
	SelectRenewalCertificate([]CertStat) (CertStat, bool)
}
