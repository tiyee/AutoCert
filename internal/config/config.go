package config

type ICredentials struct {
	AccessKeyId     string
	AccessKeySecret string
}
type Config struct {
	Platform        string       `yaml:"platform"`
	DNSPlatform        string    `yaml:"dns_platform"`
	CDNPlatform        string    `yaml:"cdn_platform"`
	CertOnly        bool         `yaml:"cert_only"`
	Renew           bool         `yaml:"renew"`
	Domain          string       `yaml:"domain"`
	Product         string       `yaml:"product"`
	Email           string       `yaml:"email"`
	DNSCredentials  ICredentials `yaml:"dnsCredentials"`
	CertCredentials ICredentials `yaml:"certCredentials"`
}
