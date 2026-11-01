package AutoCert

import (
	"context"
	"fmt"

	"github.com/tiyee/AutoCert/internal/applicant"
	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func newApplicant(cfg config.Config) (applicant.IApplicant, error) {
	af := applicant.GetApplicant(cfg.DNSPlatform)
	if af == nil {
		return nil, fmt.Errorf("apply platform %q not supported", cfg.DNSPlatform)
	}
	return af(applicant.WithDomain(cfg.Domain), applicant.WithEmail(cfg.Email), applicant.WithAccessKeyId(cfg.DNSCredentials.AccessKeyId), applicant.WithAccessKeySecret(cfg.DNSCredentials.AccessKeySecret), applicant.WithSecurityToken(cfg.DNSCredentials.SecurityToken), applicant.WithSiteID(cfg.DNSSiteID), applicant.WithRegion(cfg.DNSRegion), applicant.WithNameservers("1.1.1.1;8.8.8.8")), nil
}
func newDeployer(cfg config.Config) (deployer.IDeployer, error) {
	df := deployer.GetDeployer(cfg.CDNPlatform)
	if df == nil {
		return nil, fmt.Errorf("deploy platform %q not supported", cfg.CDNPlatform)
	}
	return df(deployer.WithDomain(cfg.Domain), deployer.WithAccessKeyId(cfg.CertCredentials.AccessKeyId), deployer.WithAccessKeySecret(cfg.CertCredentials.AccessKeySecret), deployer.WithSecurityToken(cfg.CertCredentials.SecurityToken), deployer.WithSiteID(cfg.CDNSiteID), deployer.WithRegion(cfg.CDNRegion), deployer.WithVariables(make(map[string]string)))
}
func reportSuccess(platform string) {
	if platform == "esa" || platform == "teo" {
		fmt.Println("Certificate deployment submitted successfully")
	} else if platform == "tencent" {
		fmt.Println("Successfully uploaded; CDN binding is not performed")
	} else {
		fmt.Println("Successfully deployed")
	}
}
func CertOnly(cfg config.Config) error {
	cfg = cfg.WithDefaults()
	applyClient, err := newApplicant(cfg)
	if err != nil {
		return err
	}
	// 在申请之前检查部署端，避免签发后才发现平台不支持。
	sslClient, err := newDeployer(cfg)
	if err != nil {
		return err
	}
	if preflight, ok := sslClient.(deployer.PreflightDeployer); ok {
		if err := preflight.Preflight(context.Background()); err != nil {
			return fmt.Errorf("deployment preflight: %w", err)
		}
	}
	certificate, err := applyClient.Apply()
	if err != nil {
		return fmt.Errorf("apply: %w", err)
	}
	if certificate == nil {
		return fmt.Errorf("apply returned no certificate")
	}
	sslClient.SetCertificate(*certificate)
	if err = sslClient.Deploy(context.Background()); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	reportSuccess(cfg.CDNPlatform)
	return nil
}
