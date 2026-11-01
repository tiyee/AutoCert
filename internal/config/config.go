package config

type ICredentials struct {
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
}
type Config struct {
	DNSSiteID       string       `yaml:"dns_site_id"`
	CDNSiteID       string       `yaml:"cdn_site_id"`
	DNSRegion       string       `yaml:"dns_region"`
	CDNRegion       string       `yaml:"cdn_region"`
	Platform        string       `yaml:"platform"`
	DNSPlatform     string       `yaml:"dns_platform"`
	CDNPlatform     string       `yaml:"cdn_platform"`
	CertOnly        bool         `yaml:"cert_only"`
	Renew           bool         `yaml:"renew"`
	Domain          string       `yaml:"domain"`
	Product         string       `yaml:"product"`
	Email           string       `yaml:"email"`
	DNSCredentials  ICredentials `yaml:"dnsCredentials"`
	CertCredentials ICredentials `yaml:"certCredentials"`
}

// WithDefaults 保持旧 platform 参数对 DNS/CDN 的回退行为。
func (c Config) WithDefaults() Config {
	if c.Platform == "" {
		c.Platform = "aliyun"
	}
	if c.DNSPlatform == "" {
		c.DNSPlatform = c.Platform
	}
	if c.CDNPlatform == "" {
		c.CDNPlatform = c.Platform
	}
	return c
}
