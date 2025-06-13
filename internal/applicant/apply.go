package applicant

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

type Certificate struct {
	CertUrl           string `json:"certUrl"`
	CertStableUrl     string `json:"certStableUrl"`
	PrivateKey        string `json:"privateKey"`
	Certificate       string `json:"certificate"`
	IssuerCertificate string `json:"issuerCertificate"`
	Csr               string `json:"csr"`
}

type IApplicant interface {
	Apply() (*Certificate, error)
}
type applicantFactory = func(option ...Option) IApplicant

var applicantMap = map[string]applicantFactory{}

func GetApplicant(platform string) applicantFactory {
	return applicantMap[platform]
}

func RegisterApplicant(platform string, f applicantFactory) {
	applicantMap[platform] = f
}

type IUser interface {
	registration.User
	SetEmail(email string)
	SetPrivateKey(privateKey crypto.PrivateKey)
	SetRegistration(registration *registration.Resource)
}

func apply(option *Options, provider challenge.Provider) (*Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	user := &User{}
	user.SetPrivateKey(privateKey)
	user.SetEmail(option.Email)
	config := lego.NewConfig(user)
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	challengeOptions := make([]dns01.ChallengeOption, 0)
	nameservers := ParseNameservers(option.Nameservers)
	if len(nameservers) > 0 {
		challengeOptions = append(challengeOptions,
			dns01.AddRecursiveNameservers(nameservers),
			dns01.PropagationWait(time.Second, true))
	}

	if err := client.Challenge.SetDNS01Provider(provider, challengeOptions...); err != nil {
		return nil, err
	}
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}

	user.SetRegistration(reg)

	domains := []string{option.Domain}

	// 如果是通配置符域名，把根域名也加入
	if strings.HasPrefix(option.Domain, "*.") && len(strings.Split(option.Domain, ".")) == 3 {
		rootDomain := strings.TrimPrefix(option.Domain, "*.")
		domains = append(domains, rootDomain)
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		CertUrl:           certificates.CertURL,
		CertStableUrl:     certificates.CertStableURL,
		PrivateKey:        string(certificates.PrivateKey),
		Certificate:       string(certificates.Certificate),
		IssuerCertificate: string(certificates.IssuerCertificate),
		Csr:               string(certificates.CSR),
	}, nil
}

func ParseNameservers(ns string) []string {
	nameservers := make([]string, 0)

	lines := strings.Split(ns, ";")

	for _, line := range lines {

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		nameservers = append(nameservers, line)
	}

	return nameservers
}
