package acme

import (
	"github.com/neko233-com/acme233/internal/config"
	"github.com/neko233-com/acme233/internal/deploy"
)

// certificateRuntime is the resolved execution view for one certificate.
//
// Config loading keeps the declarative shape intact, while runtime code only
// consumes this flattened view so issue/revoke/install/deploy all share the
// same provider, derived env, and deploy context semantics.
type certificateRuntime struct {
	cert          config.CertificateSpec
	dns           config.DNSConfig
	deployContext deploy.Context
}

func resolveCertificateRuntime(cfg *config.Config, cert config.CertificateSpec) (certificateRuntime, error) {
	effectiveDNS, err := cfg.EffectiveDNS(cert)
	if err != nil {
		return certificateRuntime{}, err
	}

	paths := cert.Paths()
	return certificateRuntime{
		cert: cert,
		dns:  effectiveDNS,
		deployContext: deploy.Context{
			Name:         cert.Name,
			OutputDir:    cert.OutputDir,
			CertFile:     paths.CertFile,
			KeyFile:      paths.KeyFile,
			PublicKey:    paths.PublicKeyFile,
			FullChain:    paths.FullChainFile,
			ChainFile:    paths.ChainFile,
			MetadataFile: paths.MetadataFile,
			Provider:     effectiveDNS.Provider,
			Challenge:    cert.Challenge,
			Domain:       cert.Domains[0],
			DomainsCSV:   joinDomains(cert.Domains),
			DirectoryURL: cfg.CA.DirectoryURL,
		},
	}, nil
}

// withChallengeEnv scopes provider env mutation to the provider construction
// window only. lego DNS providers read credentials from process env, so this
// keeps the rest of the workflow free from shared mutable state.
func (r certificateRuntime) withChallengeEnv(run func() error) error {
	if r.cert.Challenge != "dns-01" {
		return run()
	}

	restoreEnv, err := applyEnv(r.dns)
	if err != nil {
		return err
	}
	defer restoreEnv()

	return run()
}

func joinDomains(domains []string) string {
	if len(domains) == 0 {
		return ""
	}
	if len(domains) == 1 {
		return domains[0]
	}

	size := 0
	for _, domain := range domains {
		size += len(domain)
	}
	builder := make([]byte, 0, size+len(domains)-1)
	for index, domain := range domains {
		if index > 0 {
			builder = append(builder, ',')
		}
		builder = append(builder, domain...)
	}
	return string(builder)
}
