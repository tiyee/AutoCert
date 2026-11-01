package AutoCert

import (
	"context"
	"fmt"

	"github.com/tiyee/AutoCert/internal/config"
	"github.com/tiyee/AutoCert/internal/deployer"
)

func Renew(cfg config.Config) error {
	cfg = cfg.WithDefaults()
	applyClient, err := newApplicant(cfg)
	if err != nil {
		return err
	}
	sslClient, err := newDeployer(cfg)
	if err != nil {
		return err
	}
	certs, err := sslClient.Search(cfg.Domain)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	selected, ok := deployer.LatestCertificate(certs)
	if selector, hasSelector := sslClient.(deployer.RenewalSelector); hasSelector {
		selected, ok = selector.SelectRenewalCertificate(certs)
	}
	if !ok {
		return fmt.Errorf("no certs found; use --certonly for initial issuance")
	}
	if selected.LastTime <= 0 {
		return fmt.Errorf("certificate has no valid expiration time")
	}
	if !selected.Renewable() {
		fmt.Println("Certificate no need renew")
		return nil
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
