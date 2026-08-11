package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"
	"time"

	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
)

func TestVerifyRsaCertificateEnforcesGeneration1Invariants(t *testing.T) {
	verificationTime := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	caKey, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // Appendix 11 mandates RSA-1024 for Gen1.
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	const caReference uint64 = 18250066869723594497
	root := rsaRootCertificate(caKey, caReference)

	valid := rsaCertificateContent{
		profile:   0x01,
		car:       caReference,
		cha:       [7]byte{0xff, 0x54, 0x41, 0x43, 0x48, 0x4f, byte(RsaCertificateRoleVehicleUnit)},
		expiresAt: verificationTime.Add(time.Hour),
		chr:       1316820541096591105,
	}
	cert := signedRsaCertificate(t, caKey, valid, caReference)
	if err := VerifyRsaCertificateWithRootAt(cert, root, verificationTime); err != nil {
		t.Fatalf("VerifyRsaCertificateWithRootAt() error = %v", err)
	}
	if !cert.GetSignatureValid() {
		t.Fatal("signature_valid = false, want true")
	}
	if cert.GetCertificateProfileIdentifier() != 1 {
		t.Fatalf("certificate_profile_identifier = %d, want 1", cert.GetCertificateProfileIdentifier())
	}
	if err := VerifyRsaCertificateRole(cert, RsaCertificateRoleVehicleUnit); err != nil {
		t.Fatalf("VerifyRsaCertificateRole() error = %v", err)
	}

	tests := []struct {
		name      string
		content   rsaCertificateContent
		appended  uint64
		at        time.Time
		wantError string
	}{
		{
			name:      "wrong profile",
			content:   withRsaProfile(valid, 0x02),
			appended:  caReference,
			at:        verificationTime,
			wantError: "invalid certificate profile identifier",
		},
		{
			name:      "recovered CAR differs from appended CAR",
			content:   withRsaCAR(valid, caReference-1),
			appended:  caReference,
			at:        verificationTime,
			wantError: "does not match appended CAR'",
		},
		{
			name:      "expired",
			content:   valid,
			appended:  caReference,
			at:        valid.expiresAt.Add(time.Second),
			wantError: "certificate expired",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := signedRsaCertificate(t, caKey, tt.content, tt.appended)
			err := VerifyRsaCertificateWithRootAt(cert, root, tt.at)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("VerifyRsaCertificateWithRootAt() error = %v, want containing %q", err, tt.wantError)
			}
			if cert.GetSignatureValid() && tt.wantError != "certificate expired" {
				t.Fatal("signature_valid = true after structural certificate failure")
			}
		})
	}
}

func TestVerifyRsaCertificateRole(t *testing.T) {
	cert := &securityv1.RsaCertificate{}
	cert.SetCertificateHolderAuthorisation(
		[]byte{0xff, 0x54, 0x41, 0x43, 0x48, 0x4f, byte(RsaCertificateRoleVehicleUnit)})

	if err := VerifyRsaCertificateRole(cert, RsaCertificateRoleVehicleUnit); err != nil {
		t.Fatalf("VerifyRsaCertificateRole() error = %v", err)
	}
	if err := VerifyRsaCertificateRole(cert, RsaCertificateRoleMemberState); err == nil {
		t.Fatal("VerifyRsaCertificateRole() accepted the wrong role")
	}
	cert.SetCertificateHolderAuthorisation(
		[]byte{0, 0, 0, 0, 0, 0, byte(RsaCertificateRoleVehicleUnit)})
	if err := VerifyRsaCertificateRole(cert, RsaCertificateRoleVehicleUnit); err == nil {
		t.Fatal("VerifyRsaCertificateRole() accepted the wrong application ID")
	}
}

func TestVerifyRsaCertificateValidityAtAllowsMissingEndOfValidity(t *testing.T) {
	verificationTime := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	if err := VerifyRsaCertificateValidityAt(&securityv1.RsaCertificate{}, verificationTime); err != nil {
		t.Fatalf("VerifyRsaCertificateValidityAt() error = %v", err)
	}
}

type rsaCertificateContent struct {
	profile   byte
	car       uint64
	cha       [7]byte
	expiresAt time.Time
	chr       uint64
}

func signedRsaCertificate(t *testing.T, caKey *rsa.PrivateKey, content rsaCertificateContent, appendedCAR uint64) *securityv1.RsaCertificate {
	t.Helper()

	cPrime := make([]byte, 164)
	cPrime[0] = content.profile
	binary.BigEndian.PutUint64(cPrime[1:9], content.car)
	copy(cPrime[9:16], content.cha[:])
	if content.expiresAt.IsZero() {
		for i := 16; i < 20; i++ {
			cPrime[i] = 0xff
		}
	} else {
		binary.BigEndian.PutUint32(cPrime[16:20], uint32(content.expiresAt.Unix()))
	}
	binary.BigEndian.PutUint64(cPrime[20:28], content.chr)
	copy(cPrime[28:156], caKey.N.FillBytes(make([]byte, 128)))
	binary.BigEndian.PutUint64(cPrime[156:164], uint64(caKey.E))

	digest := sha1.Sum(cPrime) //nolint:gosec // SHA-1 is required by Appendix 11 for Gen1 certificates.
	recovered := make([]byte, 128)
	recovered[0] = 0x6a
	copy(recovered[1:107], cPrime[:106])
	copy(recovered[107:127], digest[:])
	recovered[127] = 0xbc

	signature := new(big.Int).Exp(new(big.Int).SetBytes(recovered), caKey.D, caKey.N)
	raw := make([]byte, 194)
	copy(raw[:128], signature.FillBytes(make([]byte, 128)))
	copy(raw[128:186], cPrime[106:])
	binary.BigEndian.PutUint64(raw[186:194], appendedCAR)

	cert := &securityv1.RsaCertificate{}
	cert.SetRawData(raw)
	return cert
}

func rsaRootCertificate(key *rsa.PrivateKey, keyID uint64) *securityv1.RootCertificate {
	root := &securityv1.RootCertificate{}
	root.SetKeyId(new(big.Int).SetUint64(keyID).String())
	root.SetRsaModulus(key.N.FillBytes(make([]byte, 128)))
	exponent := make([]byte, 8)
	binary.BigEndian.PutUint64(exponent, uint64(key.E))
	root.SetRsaExponent(exponent)
	return root
}

func withRsaProfile(content rsaCertificateContent, profile byte) rsaCertificateContent {
	content.profile = profile
	return content
}

func withRsaCAR(content rsaCertificateContent, car uint64) rsaCertificateContent {
	content.car = car
	return content
}
