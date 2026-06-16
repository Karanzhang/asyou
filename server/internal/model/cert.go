package model

import "time"

// Certificate stores an ACME-provisioned TLS certificate.
type Certificate struct {
	ID         int64     `db:"id" json:"id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	ProxyID    int64     `db:"proxy_id" json:"proxy_id"`
	Domain     string    `db:"domain" json:"domain"`
	CertPEM    string    `db:"cert_pem" json:"-"`
	KeyPEM     string    `db:"key_pem" json:"-"`
	Issuer     string    `db:"issuer" json:"issuer"`
	ExpiresAt  time.Time `db:"expires_at" json:"expires_at"`
	AutoRenew  bool      `db:"auto_renew" json:"auto_renew"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}
