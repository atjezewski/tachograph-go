package cert

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
)

// ChainResolver chains multiple certificate resolvers, trying each in sequence
// until one succeeds.
type ChainResolver struct {
	resolvers []Resolver
}

var _ Resolver = &ChainResolver{}

// NewChainResolver creates a new [ChainResolver].
func NewChainResolver(resolvers ...Resolver) *ChainResolver {
	return &ChainResolver{
		resolvers: resolvers,
	}
}

// GetRootCertificate implements [Resolver.GetRootCertificate].
func (r *ChainResolver) GetRootCertificate(ctx context.Context) (*securityv1.RootCertificate, error) {
	var errs []error
	for _, resolver := range r.resolvers {
		cert, err := resolver.GetRootCertificate(ctx)
		if err == nil {
			return cert, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("failed to get root certificate: %w", errors.Join(errs...))
}

// GetEccRootCertificate implements [Resolver.GetEccRootCertificate].
func (r *ChainResolver) GetEccRootCertificate(ctx context.Context) (*securityv1.EccCertificate, error) {
	var errs []error
	for _, resolver := range r.resolvers {
		cert, err := resolver.GetEccRootCertificate(ctx)
		if err == nil {
			return cert, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("failed to get Gen2 root certificate: %w", errors.Join(errs...))
}

// GetEccRootCertificates returns the union of roots exposed by the chained
// resolvers, deduplicated by certificate holder reference.
func (r *ChainResolver) GetEccRootCertificates(ctx context.Context) ([]*securityv1.EccCertificate, error) {
	var roots []*securityv1.EccCertificate
	var errs []error
	seen := make(map[string]struct{})
	for _, resolver := range r.resolvers {
		var resolved []*securityv1.EccCertificate
		var err error
		if rootSetResolver, ok := resolver.(EccRootSetResolver); ok {
			resolved, err = rootSetResolver.GetEccRootCertificates(ctx)
		} else {
			var root *securityv1.EccCertificate
			root, err = resolver.GetEccRootCertificate(ctx)
			if root != nil {
				resolved = []*securityv1.EccCertificate{root}
			}
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, root := range resolved {
			if root == nil {
				continue
			}
			key := root.GetCertificateHolderReference()
			if raw := root.GetCertificateHolderReferenceRaw(); len(raw) > 0 {
				key = string(raw)
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		if len(errs) == 0 {
			return nil, fmt.Errorf("no Gen2 root certificates found")
		}
		return nil, fmt.Errorf("failed to get Gen2 root certificates: %w", errors.Join(errs...))
	}
	return roots, nil
}

// GetRsaCertificate implements [Resolver.GetRsaCertificate].
func (r *ChainResolver) GetRsaCertificate(ctx context.Context, chr string) (*securityv1.RsaCertificate, error) {
	var errs []error
	for _, resolver := range r.resolvers {
		cert, err := resolver.GetRsaCertificate(ctx, chr)
		if err == nil {
			return cert, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("failed to get RSA certificate: %w", errors.Join(errs...))
}

// GetEccCertificate implements [Resolver.GetEccCertificate].
func (r *ChainResolver) GetEccCertificate(ctx context.Context, chr string) (*securityv1.EccCertificate, error) {
	var errs []error
	for _, resolver := range r.resolvers {
		cert, err := resolver.GetEccCertificate(ctx, chr)
		if err == nil {
			return cert, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("failed to get ECC certificate: %w", errors.Join(errs...))
}

var (
	defaultResolver     Resolver
	defaultResolverOnce sync.Once
)

// DefaultResolver returns the default certificate resolver.
//
// The default resolver uses a chain of certificate sources:
//  1. Embedded certificates from EU member states (fast, offline)
//  2. Remote certificate fetching via HTTP (fallback, requires network)
//
// This resolver is suitable for most use cases and provides good performance
// while ensuring compatibility with certificates from all EU member states.
//
// The resolver is initialized lazily and cached for subsequent calls.
func DefaultResolver() Resolver {
	defaultResolverOnce.Do(func() {
		defaultResolver = NewChainResolver(
			NewEmbeddedResolver(),
			NewClient(http.DefaultClient),
		)
	})
	return defaultResolver
}
