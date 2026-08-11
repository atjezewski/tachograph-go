package tachograph

import (
	"github.com/way-platform/tachograph-go/internal/cert"
)

// CertificateResolver provides access to tachograph certificates needed for
// signature verification.
//
// Implementations of this interface are responsible for fetching certificates
// by their Certificate Holder Reference (CHR). The default implementation uses
// embedded certificates from EU member states and falls back to fetching from
// remote sources.
type CertificateResolver = cert.Resolver

// EccRootSetResolver is an optional capability for certificate resolvers that
// retain multiple trusted Gen2 ERCA root generations. Authentication falls
// back to CertificateResolver.GetEccRootCertificate when it is not implemented.
type EccRootSetResolver = cert.EccRootSetResolver

// DefaultCertificateResolver returns the default certificate resolver.
//
// The default resolver uses a chain of certificate sources:
//  1. Embedded certificates from EU member states (fast, offline)
//  2. Remote certificate fetching via HTTP (fallback, requires network)
//
// This resolver is suitable for most use cases and provides good performance
// while ensuring compatibility with certificates from all EU member states.
//
// The resolver is initialized lazily and cached for subsequent calls.
func DefaultCertificateResolver() CertificateResolver {
	return cert.DefaultResolver()
}
