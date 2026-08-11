package cert

import (
	"context"

	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
)

// Resolver is an interface for resolving tachograph certificates.
type Resolver interface {
	// GetRootCertificate retrieves the European Root CA certificate.
	GetRootCertificate(ctx context.Context) (*securityv1.RootCertificate, error)

	// GetEccRootCertificate retrieves the Gen2 European Root CA certificate (ECC).
	GetEccRootCertificate(ctx context.Context) (*securityv1.EccCertificate, error)

	// GetRsaCertificate retrieves an RSA certificate (Generation 1) by its CHR.
	GetRsaCertificate(ctx context.Context, chr string) (*securityv1.RsaCertificate, error)

	// GetEccCertificate retrieves an ECC certificate (Generation 2) by its CHR.
	GetEccCertificate(ctx context.Context, chr string) (*securityv1.EccCertificate, error)
}

// EccRootSetResolver optionally exposes every trusted Gen2 ERCA root that may
// anchor a certificate chain. Resolver remains intentionally unchanged so
// existing custom resolvers continue to work.
type EccRootSetResolver interface {
	GetEccRootCertificates(ctx context.Context) ([]*securityv1.EccCertificate, error)
}
