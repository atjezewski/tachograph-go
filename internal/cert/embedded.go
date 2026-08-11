package cert

import (
	"context"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/cert/certcache"
	"github.com/way-platform/tachograph-go/internal/security"
	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
)

// EmbeddedResolver resolves certificates from the embedded certificate cache.
type EmbeddedResolver struct{}

var _ Resolver = &EmbeddedResolver{}

// NewEmbeddedResolver creates a new [EmbeddedResolver].
func NewEmbeddedResolver() *EmbeddedResolver {
	return &EmbeddedResolver{}
}

// GetRootCertificate retrieves the European Root CA certificate.
func (r *EmbeddedResolver) GetRootCertificate(ctx context.Context) (*securityv1.RootCertificate, error) {
	return Root()
}

// GetEccRootCertificate retrieves the Gen2 European Root CA certificate.
func (r *EmbeddedResolver) GetEccRootCertificate(ctx context.Context) (*securityv1.EccCertificate, error) {
	return RootGen2()
}

// GetEccRootCertificates retrieves all embedded Gen2 European Root CA certificates.
func (r *EmbeddedResolver) GetEccRootCertificates(ctx context.Context) ([]*securityv1.EccCertificate, error) {
	root, err := RootGen2()
	if err != nil {
		return nil, err
	}
	return []*securityv1.EccCertificate{root}, nil
}

// GetRsaCertificate retrieves an RSA certificate by its CHR.
func (r *EmbeddedResolver) GetRsaCertificate(ctx context.Context, chr string) (*securityv1.RsaCertificate, error) {
	data, ok := certcache.ReadG1(chr)
	if !ok {
		return nil, fmt.Errorf("certificate not found in embedded cache: CHR %s", chr)
	}
	return security.UnmarshalRsaCertificate(data)
}

// GetEccCertificate retrieves an ECC certificate by its CHR.
func (r *EmbeddedResolver) GetEccCertificate(ctx context.Context, chr string) (*securityv1.EccCertificate, error) {
	data, ok := certcache.ReadG2(chr)
	if !ok {
		return nil, fmt.Errorf("certificate not found in embedded cache: CHR %s", chr)
	}
	return security.UnmarshalEccCertificate(data)
}
