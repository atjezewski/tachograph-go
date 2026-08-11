package card

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
	"google.golang.org/protobuf/proto"
)

// TestAuthenticateCardCorpus exercises complete local DDD files without
// committing sensitive fixtures. Set TACHOGRAPH_CARD_AUTH_FIXTURES to a
// path-list containing known-good Gen1, Gen2 V1, and Gen2 V2 downloads.
func TestAuthenticateCardCorpus(t *testing.T) {
	fixtureList := os.Getenv("TACHOGRAPH_CARD_AUTH_FIXTURES")
	if fixtureList == "" {
		t.Skip("TACHOGRAPH_CARD_AUTH_FIXTURES is not set")
	}

	for _, path := range filepath.SplitList(fixtureList) {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("os.ReadFile() error = %v", err)
			}
			rawFile, err := (UnmarshalOptions{Strict: true}).UnmarshalRawCardFile(data)
			if err != nil {
				t.Fatalf("UnmarshalRawCardFile() error = %v", err)
			}
			if err := (AuthenticateOptions{}).AuthenticateRawCardFile(context.Background(), rawFile); err != nil {
				t.Fatalf("baseline authentication error = %v", err)
			}
			assertCardAuthenticationMetadata(t, rawFile)
			if value := os.Getenv("TACHOGRAPH_CARD_AUTH_VERIFICATION_TIME"); value != "" {
				verificationTime, err := time.Parse(time.RFC3339, value)
				if err != nil {
					t.Fatalf("invalid TACHOGRAPH_CARD_AUTH_VERIFICATION_TIME: %v", err)
				}
				timed := proto.Clone(rawFile).(*cardv1.RawCardFile)
				if err := (AuthenticateOptions{VerificationTime: verificationTime}).AuthenticateRawCardFile(context.Background(), timed); err != nil {
					t.Fatalf("authentication at %s error = %v", verificationTime, err)
				}
				assertCardVerificationTime(t, timed, verificationTime)
			}

			t.Run("expired certificate", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*cardv1.RawCardFile)
				assertCardAuthenticationFailureWithOptions(
					t,
					mutated,
					latestCardGeneration(mutated),
					securityv1.Authentication_CERTIFICATE_VERIFICATION_FAILED,
					"expired",
					AuthenticateOptions{VerificationTime: time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC)},
				)
			})

			t.Run("certificate", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*cardv1.RawCardFile)
				record := cardCertificateRecord(t, mutated)
				value := append([]byte(nil), record.GetValue()...)
				value[len(value)/2] ^= 0x01
				record.SetValue(value)
				assertCardAuthenticationFailure(t, mutated, record.GetGeneration(), securityv1.Authentication_CERTIFICATE_VERIFICATION_FAILED, "certificate")
			})

			t.Run("signed data", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*cardv1.RawCardFile)
				record, _ := signedCardRecordPair(t, mutated)
				value := append([]byte(nil), record.GetValue()...)
				value[len(value)-1] ^= 0x01
				record.SetValue(value)
				assertCardAuthenticationFailure(t, mutated, record.GetGeneration(), securityv1.Authentication_DATA_SIGNATURE_INVALID, "signature verification failed")
			})

			t.Run("signature", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*cardv1.RawCardFile)
				dataRecord, signatureRecord := signedCardRecordPair(t, mutated)
				value := append([]byte(nil), signatureRecord.GetValue()...)
				value[len(value)-1] ^= 0x01
				signatureRecord.SetValue(value)
				assertCardAuthenticationFailure(t, mutated, dataRecord.GetGeneration(), securityv1.Authentication_DATA_SIGNATURE_INVALID, "signature verification failed")
			})
		})
	}
}

func cardCertificateRecord(t *testing.T, rawFile *cardv1.RawCardFile) *cardv1.RawCardFile_Record {
	t.Helper()
	wantGeneration := latestCardGeneration(rawFile)
	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() != wantGeneration {
			continue
		}
		if record.GetFile() == cardv1.ElementaryFileType_EF_CARD_CERTIFICATE ||
			record.GetFile() == cardv1.ElementaryFileType_EF_CARD_SIGN_CERTIFICATE {
			return record
		}
	}
	t.Fatal("card certificate record not found")
	return nil
}

func signedCardRecordPair(t *testing.T, rawFile *cardv1.RawCardFile) (*cardv1.RawCardFile_Record, *cardv1.RawCardFile_Record) {
	t.Helper()
	wantGeneration := latestCardGeneration(rawFile)
	records := rawFile.GetRecords()
	for i, record := range records {
		if record.GetGeneration() != wantGeneration ||
			record.GetContentType() != cardv1.ContentType_DATA ||
			!isSignedEF(record.GetFile()) || i+1 >= len(records) {
			continue
		}
		signatureRecord := records[i+1]
		if signatureRecord.GetGeneration() == wantGeneration &&
			signatureRecord.GetContentType() == cardv1.ContentType_SIGNATURE &&
			signatureRecord.GetFile() == record.GetFile() {
			return record, signatureRecord
		}
	}
	t.Fatal("signed card record pair not found")
	return nil, nil
}

func latestCardGeneration(rawFile *cardv1.RawCardFile) ddv1.Generation {
	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() == ddv1.Generation_GENERATION_2 {
			return ddv1.Generation_GENERATION_2
		}
	}
	return ddv1.Generation_GENERATION_1
}

func assertCardAuthenticationMetadata(t *testing.T, rawFile *cardv1.RawCardFile) {
	t.Helper()
	for _, record := range rawFile.GetRecords() {
		if record.GetContentType() != cardv1.ContentType_DATA || !isSignedEF(record.GetFile()) {
			continue
		}
		auth := record.GetAuthentication()
		if auth.GetStatus() != securityv1.Authentication_VERIFIED {
			t.Fatalf("EF %v authentication status = %v, want VERIFIED", record.GetFile(), auth.GetStatus())
		}
		if auth.GetSignerCertificate() == nil || auth.GetRootCertificate() == nil {
			t.Fatalf("EF %v is missing certificate metadata", record.GetFile())
		}
	}
}

func assertCardVerificationTime(t *testing.T, rawFile *cardv1.RawCardFile, want time.Time) {
	t.Helper()
	for _, record := range rawFile.GetRecords() {
		if record.GetContentType() != cardv1.ContentType_DATA || !isSignedEF(record.GetFile()) {
			continue
		}
		got := record.GetAuthentication().GetSignatureCreationTime()
		if got == nil || !got.IsValid() || !got.AsTime().Equal(want) {
			t.Fatalf("EF %v verification time = %v, want %v", record.GetFile(), got, want)
		}
	}
}

func assertCardAuthenticationFailure(t *testing.T, rawFile *cardv1.RawCardFile, generation ddv1.Generation, wantStatus securityv1.Authentication_Status, wantError string) {
	t.Helper()
	assertCardAuthenticationFailureWithOptions(t, rawFile, generation, wantStatus, wantError, AuthenticateOptions{})
}

func assertCardAuthenticationFailureWithOptions(t *testing.T, rawFile *cardv1.RawCardFile, generation ddv1.Generation, wantStatus securityv1.Authentication_Status, wantError string, opts AuthenticateOptions) {
	t.Helper()
	err := opts.AuthenticateRawCardFile(context.Background(), rawFile)
	if err == nil || !strings.Contains(err.Error(), wantError) {
		t.Fatalf("AuthenticateRawCardFile() error = %v, want containing %q", err, wantError)
	}
	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() == generation && record.GetAuthentication().GetStatus() == wantStatus {
			return
		}
	}
	t.Fatalf("no Generation %v record has authentication status %v", generation, wantStatus)
}
