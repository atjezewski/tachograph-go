package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/way-platform/tachograph-go/internal/security"
	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestResolveEccTrustAnchorSelectsRootByMSCAReference(t *testing.T) {
	verificationTime := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	oldKey := newEccKey(t)
	newKey := newEccKey(t)
	mscaKey := newEccKey(t)
	oldReference := []byte{1, 0, 0, 0, 0, 0, 0, 1}
	newReference := []byte{2, 0, 0, 0, 0, 0, 0, 1}
	mscaReference := []byte{3, 0, 0, 0, 0, 0, 0, 1}
	oldRoot := signedTestCertificate(t, oldKey, oldKey, oldReference, oldReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 1)
	newRoot := signedTestCertificate(t, newKey, newKey, newReference, newReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 2)
	msca := signedTestCertificate(t, newKey, mscaKey, newReference, mscaReference, security.EccCertificateRoleMemberStateCA, verificationTime, 3)
	resolver := &testRootSetResolver{testResolver: testResolver{root: oldRoot}, roots: []*securityv1.EccCertificate{oldRoot, newRoot}}

	got, err := ResolveEccTrustAnchor(context.Background(), resolver, msca, nil, verificationTime)
	if err != nil {
		t.Fatalf("ResolveEccTrustAnchor() error = %v", err)
	}
	if got != newRoot {
		t.Fatal("ResolveEccTrustAnchor() did not select the root referenced by the MSCA")
	}
}

func TestResolveEccTrustAnchorUsesLinkFromTrustedPreviousRoot(t *testing.T) {
	verificationTime := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	oldKey := newEccKey(t)
	newKey := newEccKey(t)
	mscaKey := newEccKey(t)
	oldReference := []byte{1, 0, 0, 0, 0, 0, 0, 1}
	newReference := []byte{2, 0, 0, 0, 0, 0, 0, 1}
	mscaReference := []byte{3, 0, 0, 0, 0, 0, 0, 1}
	oldRoot := signedTestCertificate(t, oldKey, oldKey, oldReference, oldReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 1)
	link := signedTestCertificate(t, oldKey, newKey, oldReference, newReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 2)
	msca := signedTestCertificate(t, newKey, mscaKey, newReference, mscaReference, security.EccCertificateRoleMemberStateCA, verificationTime, 3)
	resolver := &testResolver{root: oldRoot}

	anchor, err := ResolveEccTrustAnchor(context.Background(), resolver, msca, link, verificationTime)
	if err != nil {
		t.Fatalf("ResolveEccTrustAnchor() error = %v", err)
	}
	if anchor != link {
		t.Fatal("ResolveEccTrustAnchor() did not return the validated link certificate")
	}
	if err := security.VerifyEccCertificateWithCAAt(msca, anchor, verificationTime); err != nil {
		t.Fatalf("MSCA verification through link certificate error = %v", err)
	}
}

func TestResolveEccTrustAnchorRejectsInvalidLink(t *testing.T) {
	verificationTime := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	oldKey := newEccKey(t)
	newKey := newEccKey(t)
	mscaKey := newEccKey(t)
	oldReference := []byte{1, 0, 0, 0, 0, 0, 0, 1}
	newReference := []byte{2, 0, 0, 0, 0, 0, 0, 1}
	mscaReference := []byte{3, 0, 0, 0, 0, 0, 0, 1}
	oldRoot := signedTestCertificate(t, oldKey, oldKey, oldReference, oldReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 1)
	validLink := signedTestCertificate(t, oldKey, newKey, oldReference, newReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 2)
	msca := signedTestCertificate(t, newKey, mscaKey, newReference, mscaReference, security.EccCertificateRoleMemberStateCA, verificationTime, 3)

	t.Run("signature", func(t *testing.T) {
		link := proto.Clone(validLink).(*securityv1.EccCertificate)
		link.GetSignature().SetR([]byte{1})
		_, err := ResolveEccTrustAnchor(context.Background(), &testResolver{root: oldRoot}, msca, link, verificationTime)
		if err == nil || !strings.Contains(err.Error(), "link certificate verification failed") {
			t.Fatalf("ResolveEccTrustAnchor() error = %v, want link verification failure", err)
		}
	})

	t.Run("role", func(t *testing.T) {
		link := proto.Clone(validLink).(*securityv1.EccCertificate)
		link.SetCertificateHolderAuthorisation([]byte{0xff, 0x53, 0x4d, 0x52, 0x44, 0x54, byte(security.EccCertificateRoleVehicleUnitSign)})
		_, err := ResolveEccTrustAnchor(context.Background(), &testResolver{root: oldRoot}, msca, link, verificationTime)
		if err == nil || !strings.Contains(err.Error(), "invalid link certificate role") {
			t.Fatalf("ResolveEccTrustAnchor() error = %v, want link role failure", err)
		}
	})
}

func TestResolveEccTrustAnchorRejectsLinkKeyMismatch(t *testing.T) {
	verificationTime := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	oldKey := newEccKey(t)
	newKey := newEccKey(t)
	wrongNewKey := newEccKey(t)
	mscaKey := newEccKey(t)
	oldReference := []byte{1, 0, 0, 0, 0, 0, 0, 1}
	newReference := []byte{2, 0, 0, 0, 0, 0, 0, 1}
	mscaReference := []byte{3, 0, 0, 0, 0, 0, 0, 1}
	oldRoot := signedTestCertificate(t, oldKey, oldKey, oldReference, oldReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 1)
	newRoot := signedTestCertificate(t, newKey, newKey, newReference, newReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 2)
	link := signedTestCertificate(t, oldKey, wrongNewKey, oldReference, newReference, security.EccCertificateRoleEuropeanRootCA, verificationTime, 3)
	msca := signedTestCertificate(t, newKey, mscaKey, newReference, mscaReference, security.EccCertificateRoleMemberStateCA, verificationTime, 4)
	resolver := &testRootSetResolver{testResolver: testResolver{root: oldRoot}, roots: []*securityv1.EccCertificate{oldRoot, newRoot}}

	_, err := ResolveEccTrustAnchor(context.Background(), resolver, msca, link, verificationTime)
	if err == nil || !strings.Contains(err.Error(), "public key does not match") {
		t.Fatalf("ResolveEccTrustAnchor() error = %v, want public-key mismatch", err)
	}
}

func TestChainResolverCombinesAndDeduplicatesEccRoots(t *testing.T) {
	rootA := &securityv1.EccCertificate{}
	rootA.SetCertificateHolderReferenceRaw([]byte{1, 0, 0, 0, 0, 0, 0, 1})
	rootB := &securityv1.EccCertificate{}
	rootB.SetCertificateHolderReferenceRaw([]byte{2, 0, 0, 0, 0, 0, 0, 1})
	resolver := NewChainResolver(
		&testRootSetResolver{testResolver: testResolver{root: rootA}, roots: []*securityv1.EccCertificate{rootA, rootB}},
		&testResolver{root: rootA},
	)

	roots, err := resolver.GetEccRootCertificates(context.Background())
	if err != nil {
		t.Fatalf("GetEccRootCertificates() error = %v", err)
	}
	if len(roots) != 2 || roots[0] != rootA || roots[1] != rootB {
		t.Fatalf("GetEccRootCertificates() = %v, want rootA and rootB", roots)
	}
}

type testResolver struct {
	root *securityv1.EccCertificate
}

func (r *testResolver) GetRootCertificate(context.Context) (*securityv1.RootCertificate, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *testResolver) GetEccRootCertificate(context.Context) (*securityv1.EccCertificate, error) {
	return r.root, nil
}

func (r *testResolver) GetRsaCertificate(context.Context, string) (*securityv1.RsaCertificate, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *testResolver) GetEccCertificate(context.Context, string) (*securityv1.EccCertificate, error) {
	return nil, fmt.Errorf("not implemented")
}

type testRootSetResolver struct {
	testResolver
	roots []*securityv1.EccCertificate
}

func (r *testRootSetResolver) GetEccRootCertificates(context.Context) ([]*securityv1.EccCertificate, error) {
	return r.roots, nil
}

func newEccKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}
	return key
}

func signedTestCertificate(t *testing.T, issuerKey, subjectKey *ecdsa.PrivateKey, issuerReference, holderReference []byte, role security.EccCertificateRole, verificationTime time.Time, bodyValue byte) *securityv1.EccCertificate {
	t.Helper()
	body := []byte{0x30, 0x03, 0x02, 0x01, bodyValue}
	rawCertificate := append([]byte{0x30, byte(len(body))}, body...)
	digest := sha256.Sum256(body)
	r, s, err := ecdsa.Sign(rand.Reader, issuerKey, digest[:])
	if err != nil {
		t.Fatalf("ecdsa.Sign() error = %v", err)
	}

	publicKey := &securityv1.EccCertificate_PublicKey{}
	publicKey.SetDomainParametersOid("1.2.840.10045.3.1.7")
	publicKey.SetPublicPointX(subjectKey.X.FillBytes(make([]byte, 32)))
	publicKey.SetPublicPointY(subjectKey.Y.FillBytes(make([]byte, 32)))
	signature := &securityv1.EccCertificate_EccSignature{}
	signature.SetR(r.Bytes())
	signature.SetS(s.Bytes())
	cert := &securityv1.EccCertificate{}
	cert.SetCertificateAuthorityReferenceRaw(append([]byte(nil), issuerReference...))
	cert.SetCertificateAuthorityReference(fmt.Sprintf("%x", issuerReference))
	cert.SetCertificateHolderReferenceRaw(append([]byte(nil), holderReference...))
	cert.SetCertificateHolderReference(fmt.Sprintf("%x", holderReference))
	cert.SetCertificateHolderAuthorisation([]byte{0xff, 0x53, 0x4d, 0x52, 0x44, 0x54, byte(role)})
	cert.SetCertificateEffectiveDate(timestamppb.New(verificationTime.Add(-time.Hour)))
	cert.SetCertificateExpirationDate(timestamppb.New(verificationTime.Add(time.Hour)))
	cert.SetPublicKey(publicKey)
	cert.SetRawData(rawCertificate)
	cert.SetSignature(signature)
	return cert
}
