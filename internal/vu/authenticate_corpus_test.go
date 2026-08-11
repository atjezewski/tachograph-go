package vu

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
	"google.golang.org/protobuf/proto"
)

// TestAuthenticateVehicleUnitCorpus exercises complete local DDD files without
// committing sensitive fixtures. Set TACHOGRAPH_VU_AUTH_FIXTURES to a
// path-list containing known-good Gen1, Gen2 V1, and Gen2 V2 downloads.
func TestAuthenticateVehicleUnitCorpus(t *testing.T) {
	fixtureList := os.Getenv("TACHOGRAPH_VU_AUTH_FIXTURES")
	if fixtureList == "" {
		t.Skip("TACHOGRAPH_VU_AUTH_FIXTURES is not set")
	}

	for _, path := range filepath.SplitList(fixtureList) {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("os.ReadFile() error = %v", err)
			}
			rawFile, err := (UnmarshalOptions{Strict: true}).UnmarshalRawVehicleUnitFile(data)
			if err != nil {
				t.Fatalf("UnmarshalRawVehicleUnitFile() error = %v", err)
			}
			if err := (AuthenticateOptions{}).AuthenticateRawVehicleUnitFile(context.Background(), rawFile); err != nil {
				t.Fatalf("baseline authentication error = %v", err)
			}

			t.Run("certificate", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*vuv1.RawVehicleUnitFile)
				record := overviewRecord(t, mutated)
				value := append([]byte(nil), record.GetValue()...)
				switch record.GetGeneration() {
				case ddv1.Generation_GENERATION_1:
					value[1] ^= 0x01
				case ddv1.Generation_GENERATION_2:
					value[5] ^= 0x01
				default:
					t.Fatalf("unsupported Overview generation %v", record.GetGeneration())
				}
				record.SetValue(value)
				assertVehicleUnitAuthenticationFailure(t, mutated, securityv1.Authentication_CERTIFICATE_VERIFICATION_FAILED, "certificate")
			})

			t.Run("signed data", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*vuv1.RawVehicleUnitFile)
				record := firstNonOverviewRecord(t, mutated)
				value := append([]byte(nil), record.GetValue()...)
				value[0] ^= 0x01
				record.SetValue(value)
				assertVehicleUnitAuthenticationFailure(t, mutated, securityv1.Authentication_DATA_SIGNATURE_INVALID, "data signature verification failed")
			})

			t.Run("signature", func(t *testing.T) {
				mutated := proto.Clone(rawFile).(*vuv1.RawVehicleUnitFile)
				record := firstNonOverviewRecord(t, mutated)
				value := append([]byte(nil), record.GetValue()...)
				value[len(value)-1] ^= 0x01
				record.SetValue(value)
				assertVehicleUnitAuthenticationFailure(t, mutated, securityv1.Authentication_DATA_SIGNATURE_INVALID, "data signature verification failed")
			})
		})
	}
}

func overviewRecord(t *testing.T, rawFile *vuv1.RawVehicleUnitFile) *vuv1.RawVehicleUnitFile_Record {
	t.Helper()
	for _, record := range rawFile.GetRecords() {
		switch record.GetType() {
		case vuv1.TransferType_OVERVIEW_GEN1, vuv1.TransferType_OVERVIEW_GEN2_V1, vuv1.TransferType_OVERVIEW_GEN2_V2:
			return record
		}
	}
	t.Fatal("no Overview record found")
	return nil
}

func firstNonOverviewRecord(t *testing.T, rawFile *vuv1.RawVehicleUnitFile) *vuv1.RawVehicleUnitFile_Record {
	t.Helper()
	for _, record := range rawFile.GetRecords() {
		switch record.GetType() {
		case vuv1.TransferType_OVERVIEW_GEN1, vuv1.TransferType_OVERVIEW_GEN2_V1, vuv1.TransferType_OVERVIEW_GEN2_V2:
			continue
		}
		if record.GetGeneration() == ddv1.Generation_GENERATION_1 || record.GetGeneration() == ddv1.Generation_GENERATION_2 {
			if len(record.GetValue()) > int(record.GetSignatureSize()) {
				return record
			}
		}
	}
	t.Fatal("no signed non-Overview record found")
	return nil
}

func assertVehicleUnitAuthenticationFailure(t *testing.T, rawFile *vuv1.RawVehicleUnitFile, wantStatus securityv1.Authentication_Status, wantError string) {
	t.Helper()
	err := (AuthenticateOptions{}).AuthenticateRawVehicleUnitFile(context.Background(), rawFile)
	if err == nil || !strings.Contains(err.Error(), wantError) {
		t.Fatalf("AuthenticateRawVehicleUnitFile() error = %v, want containing %q", err, wantError)
	}
	for _, record := range rawFile.GetRecords() {
		if record.GetAuthentication().GetStatus() == wantStatus {
			return
		}
	}
	t.Fatalf("no record has authentication status %v", wantStatus)
}
