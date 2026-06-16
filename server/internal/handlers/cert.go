package handlers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"golang.org/x/crypto/acme"
)

// ACMEConfig holds settings for automatic certificate provisioning.
type ACMEConfig struct {
	DirectoryURL string // Let's Encrypt: "https://acme-v02.api.letsencrypt.org/directory"
	Email        string // Contact email for certificate notifications
	Staging      bool   // Use staging environment for testing
}

// DefaultACMEConfig returns a default ACME config.
func DefaultACMEConfig() *ACMEConfig {
	return &ACMEConfig{
		DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
		Email:        "admin@asyou.dev",
		Staging:      false,
	}
}

// ACMEClient wraps the Go ACME library for certificate provisioning.
type ACMEClient struct {
	Config *ACMEConfig
	client *acme.Client
	ctx    context.Context
}

// NewACMEClient creates an ACME client.
func NewACMEClient(cfg *ACMEConfig) *ACMEClient {
	return &ACMEClient{
		Config: cfg,
		client: &acme.Client{DirectoryURL: cfg.DirectoryURL, UserAgent: "asyou-server/0.1"},
		ctx:    context.Background(),
	}
}

// ProvisionCert attempts to provision a certificate via ACME for the given domain.
func (a *ACMEClient) ProvisionCert(domain string) (certPEM, keyPEM string, expiresAt time.Time, err error) {
	// Register account
	acct := &acme.Account{Contact: []string{"mailto:" + a.Config.Email}}
	acct, err = a.client.Register(a.ctx, acct, acme.AcceptTOS)
	if err != nil {
		// already registered — continue
	}

	// Create order
	order, err := a.client.AuthorizeOrder(a.ctx, []acme.AuthzID{
		{Type: "dns", Value: domain},
	})
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("authorize: %w", err)
	}

	// Fulfill HTTP-01 challenges
	for _, authURL := range order.AuthzURLs {
		auth, err := a.client.GetAuthorization(a.ctx, authURL)
		if err != nil {
			return "", "", time.Time{}, fmt.Errorf("get auth: %w", err)
		}
		var chal *acme.Challenge
		for _, c := range auth.Challenges {
			if c.Type == "http-01" {
				chal = c
				break
			}
		}
		if chal == nil {
			return "", "", time.Time{}, fmt.Errorf("no http-01 challenge for %s", domain)
		}
		if _, err := a.client.Accept(a.ctx, chal); err != nil {
			return "", "", time.Time{}, fmt.Errorf("accept challenge: %w", err)
		}
		if _, err := a.client.WaitAuthorization(a.ctx, authURL); err != nil {
			return "", "", time.Time{}, fmt.Errorf("wait auth: %w", err)
		}
	}

	// Generate certificate key and CSR
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate cert key: %w", err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: []string{domain}}, certKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("create CSR: %w", err)
	}

	// Finalize and fetch certificate
	certChain, _, err := a.client.CreateOrderCert(a.ctx, order.URI, csr, false)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("create cert: %w", err)
	}
	for _, der := range certChain {
		certPEM += string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	}

	keyBytes, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("marshal key: %w", err)
	}
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}))

	if len(certChain) > 0 {
		cert, err := x509.ParseCertificate(certChain[0])
		if err == nil {
			expiresAt = cert.NotAfter
		}
	}

	return certPEM, keyPEM, expiresAt, nil
}

// CertExpirySoon checks if a certificate expires within the given duration.
func CertExpirySoon(expiresAt time.Time, within time.Duration) bool {
	return time.Until(expiresAt) < within
}
