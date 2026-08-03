package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVerifyEccCertificateWithCAEnforcesIssuerAndSetsSignatureState(t *testing.T) {
	verificationTime := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	cert, ca := signedEccCertificate(t, verificationTime)

	if err := VerifyEccCertificateWithCAAt(cert, ca, verificationTime); err != nil {
		t.Fatalf("VerifyEccCertificateWithCAAt() error = %v", err)
	}
	if !cert.GetSignatureValid() {
		t.Fatal("signature_valid = false, want true")
	}

	cert.SetCertificateAuthorityReferenceRaw([]byte{9, 9, 9, 9, 9, 9, 9, 9})
	err := VerifyEccCertificateWithCA(cert, ca)
	if err == nil || !strings.Contains(err.Error(), "CAR does not match issuer CHR") {
		t.Fatalf("issuer mismatch error = %v", err)
	}
	if cert.GetSignatureValid() {
		t.Fatal("signature_valid = true after failed verification")
	}
}

func TestVerifyEccCertificateValidityAt(t *testing.T) {
	effective := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	expiration := effective.Add(24 * time.Hour)
	cert := &securityv1.EccCertificate{}
	cert.SetCertificateEffectiveDate(timestamppb.New(effective))
	cert.SetCertificateExpirationDate(timestamppb.New(expiration))

	tests := []struct {
		name    string
		at      time.Time
		wantErr bool
	}{
		{name: "before effective", at: effective.Add(-time.Second), wantErr: true},
		{name: "at effective", at: effective},
		{name: "at expiration", at: expiration},
		{name: "after expiration", at: expiration.Add(time.Second), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyEccCertificateValidityAt(cert, tt.at)
			if (err != nil) != tt.wantErr {
				t.Fatalf("VerifyEccCertificateValidityAt() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyEccCertificateRole(t *testing.T) {
	cert := &securityv1.EccCertificate{}
	cert.SetCertificateHolderAuthorisation(
		[]byte{0xff, 0x53, 0x4d, 0x52, 0x44, 0x54, byte(EccCertificateRoleVehicleUnitSign)})

	if err := VerifyEccCertificateRole(cert, EccCertificateRoleVehicleUnitSign); err != nil {
		t.Fatalf("VerifyEccCertificateRole() error = %v", err)
	}
	if err := VerifyEccCertificateRole(cert, EccCertificateRoleMemberStateCA); err == nil {
		t.Fatal("VerifyEccCertificateRole() accepted the wrong role")
	}
	cert.SetCertificateHolderAuthorisation([]byte{0, 0, 0, 0, 0, 0, byte(EccCertificateRoleVehicleUnitSign)})
	if err := VerifyEccCertificateRole(cert, EccCertificateRoleVehicleUnitSign); err == nil {
		t.Fatal("VerifyEccCertificateRole() accepted the wrong application ID")
	}
}

func signedEccCertificate(t *testing.T, verificationTime time.Time) (*securityv1.EccCertificate, *securityv1.EccCertificate) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}
	body := []byte{0x30, 0x03, 0x02, 0x01, 0x01}
	rawCertificate := append([]byte{0x30, byte(len(body))}, body...)
	digest := sha256.Sum256(body)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		t.Fatalf("ecdsa.Sign() error = %v", err)
	}

	issuerReference := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	publicKey := &securityv1.EccCertificate_PublicKey{}
	publicKey.SetDomainParametersOid("1.2.840.10045.3.1.7")
	publicKey.SetPublicPointX(privateKey.X.FillBytes(make([]byte, 32)))
	publicKey.SetPublicPointY(privateKey.Y.FillBytes(make([]byte, 32)))
	ca := &securityv1.EccCertificate{}
	ca.SetCertificateHolderReferenceRaw(issuerReference)
	ca.SetPublicKey(publicKey)

	signature := &securityv1.EccCertificate_EccSignature{}
	signature.SetR(r.Bytes())
	signature.SetS(s.Bytes())
	cert := &securityv1.EccCertificate{}
	cert.SetCertificateAuthorityReferenceRaw(issuerReference)
	cert.SetRawData(rawCertificate)
	cert.SetSignature(signature)
	cert.SetCertificateEffectiveDate(timestamppb.New(verificationTime.Add(-time.Hour)))
	cert.SetCertificateExpirationDate(timestamppb.New(verificationTime.Add(time.Hour)))
	return cert, ca
}
