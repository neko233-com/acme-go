package acme

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/dnsprovider"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	lego "github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/http/webroot"
	"github.com/go-acme/lego/v4/registration"
)

type Mode string

const (
	ModeIssue Mode = "issue"
	ModeRenew Mode = "renew"
)

type Options struct {
	Name  string
	Force bool
	Mode  Mode
	Out   io.Writer
}

type Result struct {
	Changed int
	Skipped int
}

func Plan(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}

	for _, cert := range targets {
		needsRenew, expiry, err := certificateNeedsRenew(cert)
		if err != nil {
			fmt.Fprintf(out, "%s: issue (%v)\n", cert.Name, err)
			continue
		}
		if needsRenew {
			fmt.Fprintf(out, "%s: issue (expires %s)\n", cert.Name, expiry.Format(time.RFC3339))
			continue
		}
		fmt.Fprintf(out, "%s: skip (expires %s)\n", cert.Name, expiry.Format(time.RFC3339))
	}
	return nil
}

func Run(cfg *config.Config, options Options) (Result, error) {
	if options.Out == nil {
		options.Out = io.Discard
	}

	targets, err := selectCertificates(cfg.Certificates, options.Name)
	if err != nil {
		return Result{}, err
	}

	user, err := LoadOrCreateUser(cfg.Account.Email, cfg.Account.KeyPath)
	if err != nil {
		return Result{}, err
	}

	result := Result{}
	for _, cert := range targets {
		runtime, err := resolveCertificateRuntime(cfg, cert)
		if err != nil {
			return result, err
		}

		needsRenew, expiry, err := certificateNeedsRenew(cert)
		if err == nil && !needsRenew && !options.Force {
			fmt.Fprintf(options.Out, "%s: skip, certificate valid until %s\n", cert.Name, expiry.Format(time.RFC3339))
			result.Skipped++
			continue
		}

		var client *lego.Client
		err = runtime.withChallengeEnv(func() error {
			var buildErr error
			client, buildErr = buildClient(user, cfg, runtime)
			return buildErr
		})
		if err != nil {
			return result, err
		}
		if err := ensureRegistration(client, user, cfg.Account.AcceptTOS); err != nil {
			return result, err
		}
		if err := runPreHooks(runtime.cert, options.Mode, runtime.deployContext, options.Out); err != nil {
			return result, err
		}

		resource, err := executeCertificateOperation(client, runtime.cert, options.Mode)
		if err != nil {
			return result, fmt.Errorf("%s %s: %w", options.Mode, runtime.cert.Name, err)
		}
		if err := writeCertificate(runtime, cfg, resource); err != nil {
			return result, err
		}
		if err := runSuccessActions(runtime, options.Mode, options.Out); err != nil {
			return result, err
		}
		fmt.Fprintf(options.Out, "%s: wrote certificate to %s\n", runtime.cert.Name, runtime.cert.OutputDir)
		result.Changed++
	}

	return result, nil
}

func selectCertificates(certificates []config.CertificateSpec, name string) ([]config.CertificateSpec, error) {
	if name == "" {
		copyOf := append([]config.CertificateSpec(nil), certificates...)
		sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].Name < copyOf[j].Name })
		return copyOf, nil
	}
	for _, cert := range certificates {
		if cert.Name == name {
			return []config.CertificateSpec{cert}, nil
		}
	}
	return nil, fmt.Errorf("certificate %q not found", name)
}

func applyEnv(dnsConfig config.DNSConfig) (func(), error) {
	type restore struct {
		value   string
		present bool
	}
	// Copy first so the restore phase only touches keys we intentionally mutate.
	values := map[string]string{}
	for key, value := range dnsConfig.Env {
		values[key] = value
	}
	if dnsConfig.DisableCNAMESupport {
		values["LEGO_DISABLE_CNAME_SUPPORT"] = "true"
	}
	originals := map[string]restore{}
	for key, value := range values {
		current, present := os.LookupEnv(key)
		originals[key] = restore{value: current, present: present}
		if err := os.Setenv(key, value); err != nil {
			return nil, fmt.Errorf("set env %s: %w", key, err)
		}
	}
	return func() {
		for key, item := range originals {
			if item.present {
				_ = os.Setenv(key, item.value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	}, nil
}

func buildClient(user *User, cfg *config.Config, runtime certificateRuntime) (*lego.Client, error) {
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = cfg.CA.DirectoryURL
	legoConfig.Certificate.KeyType = mapKeyType(runtime.cert.KeyType)

	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return nil, fmt.Errorf("create ACME client: %w", err)
	}

	challengeOptions := []dns01.ChallengeOption{}
	if runtime.dns.DisableCompletePropagation {
		challengeOptions = append(challengeOptions, dns01.DisableCompletePropagationRequirement())
	}
	if len(runtime.dns.RecursiveNameservers) > 0 {
		challengeOptions = append(challengeOptions, dns01.AddRecursiveNameservers(runtime.dns.RecursiveNameservers))
	}

	if err := configureChallenge(client, runtime.cert, runtime.dns, challengeOptions); err != nil {
		return nil, err
	}
	return client, nil
}

func configureChallenge(client *lego.Client, cert config.CertificateSpec, dnsConfig config.DNSConfig, dnsOptions []dns01.ChallengeOption) error {
	switch cert.Challenge {
	case "dns-01":
		strategy, err := dnsprovider.Resolve(dnsConfig.Provider)
		if err != nil {
			return err
		}
		provider, err := strategy.NewProvider(dnsConfig)
		if err != nil {
			return fmt.Errorf("build DNS provider %q: %w", dnsConfig.Provider, err)
		}
		if err := client.Challenge.SetDNS01Provider(provider, dnsOptions...); err != nil {
			return fmt.Errorf("attach DNS provider: %w", err)
		}
	case "webroot":
		provider, err := webroot.NewHTTPProvider(cert.WebrootPath)
		if err != nil {
			return fmt.Errorf("build webroot provider: %w", err)
		}
		if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
			return fmt.Errorf("attach webroot provider: %w", err)
		}
	case "http-01", "standalone":
		if err := client.Challenge.SetHTTP01Provider(http01.NewProviderServer(cert.HTTPBind, cert.HTTPPort)); err != nil {
			return fmt.Errorf("attach standalone http provider: %w", err)
		}
	case "tls-alpn-01":
		if err := client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer(cert.TLSALPNBind, cert.TLSALPNPort)); err != nil {
			return fmt.Errorf("attach tls-alpn provider: %w", err)
		}
	default:
		return fmt.Errorf("unsupported challenge %q", cert.Challenge)
	}
	return nil
}

func ensureRegistration(client *lego.Client, user *User, acceptTOS bool) error {
	if user.Registration != nil {
		return nil
	}

	registrationResource, err := client.Registration.ResolveAccountByKey()
	if err != nil {
		registrationResource, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: acceptTOS})
		if err != nil {
			return fmt.Errorf("register account: %w", err)
		}
	}
	user.Registration = registrationResource
	if err := user.Save(); err != nil {
		return err
	}
	return nil
}

// mapKeyType keeps config-facing key names decoupled from lego constants so new
// key families can be added in one place.
func mapKeyType(value string) certcrypto.KeyType {
	switch strings.ToLower(value) {
	case "ec384":
		return certcrypto.EC384
	case "rsa2048":
		return certcrypto.RSA2048
	case "rsa4096":
		return certcrypto.RSA4096
	case "rsa8192":
		return certcrypto.RSA8192
	default:
		return certcrypto.EC256
	}
}

func bundleEnabled(cert config.CertificateSpec) bool {
	if cert.Bundle == nil {
		return true
	}
	return *cert.Bundle
}

func executeCertificateOperation(client *lego.Client, cert config.CertificateSpec, mode Mode) (*certificate.Resource, error) {
	if cert.CSRPath != "" {
		request, err := buildCSRRequest(cert)
		if err != nil {
			return nil, err
		}
		return client.Certificate.ObtainForCSR(request)
	}

	if mode == ModeRenew {
		resource, err := loadCertificateResource(cert)
		if err == nil {
			return client.Certificate.RenewWithOptions(resource, &certificate.RenewOptions{
				Bundle:         bundleEnabled(cert),
				PreferredChain: cert.PreferredChain,
			})
		}
	}

	return client.Certificate.Obtain(certificate.ObtainRequest{
		Domains:        cert.Domains,
		Bundle:         bundleEnabled(cert),
		MustStaple:     cert.MustStaple,
		PreferredChain: cert.PreferredChain,
	})
}

func buildCSRRequest(cert config.CertificateSpec) (certificate.ObtainForCSRRequest, error) {
	data, err := os.ReadFile(cert.CSRPath)
	if err != nil {
		return certificate.ObtainForCSRRequest{}, fmt.Errorf("read csr: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || !strings.Contains(block.Type, "CERTIFICATE REQUEST") {
		return certificate.ObtainForCSRRequest{}, fmt.Errorf("csr file %s is not a PEM CSR", cert.CSRPath)
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return certificate.ObtainForCSRRequest{}, fmt.Errorf("parse csr: %w", err)
	}
	return certificate.ObtainForCSRRequest{
		CSR:            csr,
		Bundle:         bundleEnabled(cert),
		PreferredChain: cert.PreferredChain,
	}, nil
}

func certificateNeedsRenew(cert config.CertificateSpec) (bool, time.Time, error) {
	expiry, err := readCertificateExpiry(cert.Paths().FullChainFile)
	if err != nil {
		return true, time.Time{}, err
	}
	renewAt := time.Now().Add(time.Duration(cert.RenewBeforeDays) * 24 * time.Hour)
	return !expiry.After(renewAt), expiry, nil
}

func readCertificateExpiry(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, fmt.Errorf("no certificate found in %s", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse certificate %s: %w", path, err)
	}
	return cert.NotAfter, nil
}

func writeCertificate(runtime certificateRuntime, cfg *config.Config, resource *certificate.Resource) error {
	if err := os.MkdirAll(runtime.cert.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create certificate dir: %w", err)
	}
	paths := runtime.cert.Paths()

	leaf, chain, err := splitCertificate(resource.Certificate)
	if err != nil {
		return err
	}
	if len(chain) == 0 && len(resource.IssuerCertificate) > 0 {
		chain = resource.IssuerCertificate
	}
	fullChain := append([]byte{}, leaf...)
	fullChain = append(fullChain, chain...)
	publicKey, err := publicKeyPEMFromCertificate(leaf)
	if err != nil {
		return err
	}

	// Keep writes deterministic so behavior is easier to reason about in tests and
	// future hooks that may watch the output directory.
	files := []struct {
		path    string
		content []byte
	}{
		{path: paths.CertFile, content: leaf},
		{path: paths.ChainFile, content: chain},
		{path: paths.FullChainFile, content: fullChain},
		{path: paths.KeyFile, content: resource.PrivateKey},
		{path: paths.PublicKeyFile, content: publicKey},
	}
	for _, file := range files {
		if len(file.content) == 0 {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(file.path), 0o755); err != nil {
			return fmt.Errorf("create certificate file dir for %s: %w", file.path, err)
		}
		if err := os.WriteFile(file.path, file.content, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", file.path, err)
		}
	}

	expiry, err := readCertificateExpiry(paths.FullChainFile)
	if err != nil {
		return err
	}

	return writeMetadata(runtime, cfg, resource, expiry)
}

func publicKeyPEMFromCertificate(data []byte) ([]byte, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("no certificate found for public key output")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate for public key output: %w", err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}), nil
}

func splitCertificate(data []byte) ([]byte, []byte, error) {
	var leaf bytes.Buffer
	var chain bytes.Buffer
	index := 0
	for len(data) > 0 {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			if index == 0 {
				if err := pem.Encode(&leaf, block); err != nil {
					return nil, nil, fmt.Errorf("encode leaf certificate: %w", err)
				}
			} else {
				if err := pem.Encode(&chain, block); err != nil {
					return nil, nil, fmt.Errorf("encode chain certificate: %w", err)
				}
			}
			index++
		}
		data = rest
	}
	if leaf.Len() == 0 {
		return nil, nil, fmt.Errorf("ACME response did not contain a leaf certificate")
	}
	return leaf.Bytes(), chain.Bytes(), nil
}

func loadCertificateResource(cert config.CertificateSpec) (certificate.Resource, error) {
	paths := cert.Paths()
	certificatePEM, err := os.ReadFile(paths.FullChainFile)
	if err != nil {
		return certificate.Resource{}, err
	}
	privateKey, err := os.ReadFile(paths.KeyFile)
	if err != nil {
		return certificate.Resource{}, err
	}
	issuerPEM, _ := os.ReadFile(paths.ChainFile)
	metadata, _ := readMetadata(cert)
	resource := certificate.Resource{
		Domain:            cert.Domains[0],
		PrivateKey:        privateKey,
		Certificate:       certificatePEM,
		IssuerCertificate: issuerPEM,
	}
	if metadata != nil {
		resource.CertURL = metadata.CertURL
		resource.CertStableURL = metadata.CertStableURL
	}
	return resource, nil
}
