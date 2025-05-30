package deployer

import (
	"context"
	"fmt"
	"github.com/tiyee/AutoCert/internal/applicant"
	"time"
)

type Option struct {
	Domain          string                `json:"domain"`
	AccessKeyId     string                `json:"accessKeyId"`
	AccessKeySecret string                `json:"accessKeySecret"`
	Certificate     applicant.Certificate `json:"certificate"`
	Variables       map[string]string     `json:"variables"`
}
type CertStat struct {
	LastTime    int64  `json:"last_time"`
	Fingerprint string `json:"fingerprint"`
	CertName    string `json:"cert_name"`
	CertId      string `json:"cert_id"`
}

func (s CertStat) Expired() bool {
	fmt.Println("last_time:", time.Unix(s.LastTime, 0).Format(time.DateTime))
	return s.LastTime < time.Now().Unix()
}
func (s CertStat) Renewable() bool {
	fmt.Println("last_time:", time.Unix(s.LastTime, 0).Format(time.DateTime))
	return s.LastTime-20*3600*24 < time.Now().Unix()
}

type IDeployer interface {
	Deploy(ctx context.Context) error
	Search(domain string) ([]CertStat, error)
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
