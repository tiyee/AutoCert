package deployer

import (
	"context"
	"fmt"
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
	fmt.Println("last_time:", time.Unix(s.LastTime, 0).Format(time.DateTime))
	return s.LastTime < time.Now().Unix()
}
func (s CertStat) Renewable() bool {
	fmt.Println("last_time:", time.Unix(s.LastTime, 0).Format(time.DateTime))
	return s.LastTime-10*3600*24 < time.Now().Unix()
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
