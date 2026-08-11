package cert

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/way-platform/tachograph-go/internal/security"
	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
)

// ResolveEccTrustAnchor returns the trusted ERCA certificate that issued the
// MSCA certificate. When the resolver does not yet contain the new ERCA root,
// a valid link certificate may bridge from a trusted preceding root.
func ResolveEccTrustAnchor(ctx context.Context, resolver Resolver, mscaCert, linkCert *securityv1.EccCertificate, verificationTime time.Time) (*securityv1.EccCertificate, error) {
	if resolver == nil {
		return nil, fmt.Errorf("certificate resolver cannot be nil")
	}
	if mscaCert == nil {
		return nil, fmt.Errorf("MSCA certificate cannot be nil")
	}

	roots, err := eccRootCertificates(ctx, resolver)
	if err != nil {
		return nil, err
	}

	if linkCert != nil {
		if err := verifyEccLinkCertificate(linkCert, roots, verificationTime); err != nil {
			return nil, fmt.Errorf("link certificate verification failed: %w", err)
		}
	}

	if root := findEccRootByReference(roots, mscaCert.GetCertificateAuthorityReferenceRaw(), mscaCert.GetCertificateAuthorityReference()); root != nil {
		if err := verifyTrustedEccRoot(root, verificationTime); err != nil {
			return nil, err
		}
		return root, nil
	}

	if linkCert != nil && certificateReferenceMatches(
		mscaCert.GetCertificateAuthorityReferenceRaw(),
		mscaCert.GetCertificateAuthorityReference(),
		linkCert.GetCertificateHolderReferenceRaw(),
		linkCert.GetCertificateHolderReference(),
	) {
		return linkCert, nil
	}

	return nil, fmt.Errorf("trusted ERCA root not found for MSCA CAR %s", mscaCert.GetCertificateAuthorityReference())
}

func eccRootCertificates(ctx context.Context, resolver Resolver) ([]*securityv1.EccCertificate, error) {
	if rootSetResolver, ok := resolver.(EccRootSetResolver); ok {
		roots, err := rootSetResolver.GetEccRootCertificates(ctx)
		if err != nil {
			return nil, err
		}
		if len(roots) == 0 {
			return nil, fmt.Errorf("certificate resolver returned no Gen2 roots")
		}
		return roots, nil
	}
	root, err := resolver.GetEccRootCertificate(ctx)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("certificate resolver returned a nil Gen2 root")
	}
	return []*securityv1.EccCertificate{root}, nil
}

func verifyEccLinkCertificate(linkCert *securityv1.EccCertificate, roots []*securityv1.EccCertificate, verificationTime time.Time) error {
	if err := security.VerifyEccCertificateRole(linkCert, security.EccCertificateRoleEuropeanRootCA); err != nil {
		return fmt.Errorf("invalid link certificate role: %w", err)
	}
	previousRoot := findEccRootByReference(
		roots,
		linkCert.GetCertificateAuthorityReferenceRaw(),
		linkCert.GetCertificateAuthorityReference(),
	)
	if previousRoot == nil {
		return fmt.Errorf("trusted preceding ERCA root not found for link CAR %s", linkCert.GetCertificateAuthorityReference())
	}
	if err := verifyTrustedEccRoot(previousRoot, verificationTime); err != nil {
		return fmt.Errorf("preceding ERCA root verification failed: %w", err)
	}
	var err error
	if verificationTime.IsZero() {
		err = security.VerifyEccCertificateWithCA(linkCert, previousRoot)
	} else {
		err = security.VerifyEccCertificateWithCAAt(linkCert, previousRoot, verificationTime)
	}
	if err != nil {
		return err
	}

	if newRoot := findEccRootByReference(
		roots,
		linkCert.GetCertificateHolderReferenceRaw(),
		linkCert.GetCertificateHolderReference(),
	); newRoot != nil && !eccPublicKeysEqual(linkCert.GetPublicKey(), newRoot.GetPublicKey()) {
		return fmt.Errorf("link certificate public key does not match trusted new ERCA root")
	}
	return nil
}

func verifyTrustedEccRoot(root *securityv1.EccCertificate, verificationTime time.Time) error {
	if err := security.VerifyEccCertificateProfile(root); err != nil {
		return fmt.Errorf("invalid root certificate profile: %w", err)
	}
	if err := security.VerifyEccCertificateRole(root, security.EccCertificateRoleEuropeanRootCA); err != nil {
		return fmt.Errorf("invalid root certificate role: %w", err)
	}
	if !certificateReferenceMatches(
		root.GetCertificateAuthorityReferenceRaw(),
		root.GetCertificateAuthorityReference(),
		root.GetCertificateHolderReferenceRaw(),
		root.GetCertificateHolderReference(),
	) {
		return fmt.Errorf("root certificate CAR does not match CHR")
	}
	// The root is trusted a priori. Its self-signature cannot add trust and may
	// be unavailable in resolvers that retain only the certified public key.
	if !verificationTime.IsZero() {
		if err := security.VerifyEccCertificateValidityAt(root, verificationTime); err != nil {
			return fmt.Errorf("root certificate validity failed: %w", err)
		}
	}
	return nil
}

func findEccRootByReference(roots []*securityv1.EccCertificate, raw []byte, value string) *securityv1.EccCertificate {
	for _, root := range roots {
		if root != nil && certificateReferenceMatches(raw, value, root.GetCertificateHolderReferenceRaw(), root.GetCertificateHolderReference()) {
			return root
		}
	}
	return nil
}

func certificateReferenceMatches(leftRaw []byte, left string, rightRaw []byte, right string) bool {
	if len(leftRaw) > 0 || len(rightRaw) > 0 {
		return len(leftRaw) == 8 && len(rightRaw) == 8 && bytes.Equal(leftRaw, rightRaw)
	}
	return left != "" && left == right
}

func eccPublicKeysEqual(left, right *securityv1.EccCertificate_PublicKey) bool {
	if left == nil || right == nil {
		return false
	}
	return left.GetDomainParametersOid() == right.GetDomainParametersOid() &&
		bytes.Equal(left.GetPublicPointX(), right.GetPublicPointX()) &&
		bytes.Equal(left.GetPublicPointY(), right.GetPublicPointY())
}
