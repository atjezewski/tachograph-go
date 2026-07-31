package vu

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
)

func TestOverview_Gen2V2(t *testing.T) {
	// Discover all matching hexdump files
	hexdumpFiles, err := findHexdumpFiles(vuv1.TransferType_OVERVIEW_GEN2_V2)
	if err != nil {
		t.Fatalf("Failed to discover hexdump files: %v", err)
	}
	if len(hexdumpFiles) == 0 {
		t.Skip("No hexdump files found for OVERVIEW_GEN2_V2")
	}

	// Run subtest for each discovered file
	for _, hexdumpPath := range hexdumpFiles {
		// Use relative path from testdata as subtest name
		relPath := strings.TrimPrefix(hexdumpPath, "testdata/records/")
		testName := strings.TrimSuffix(relPath, ".data.hexdump")

		t.Run(testName, func(t *testing.T) {
			// Read hexdump
			data, err := readHexdump(hexdumpPath)
			if err != nil {
				t.Fatalf("Failed to read hexdump: %v", err)
			}

			// Unmarshal
			overview, err := unmarshalOverviewGen2V2(data)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if overview == nil {
				t.Fatal("Unmarshal returned nil")
			}

			// Golden JSON comparison
			goldenPath := goldenJSONPath(hexdumpPath)
			loadOrCreateGolden(t, overview, goldenPath)

			// Round-trip test - marshal
			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalOverviewGen2V2(overview)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if diff := cmp.Diff(data, marshaled); diff != "" {
				t.Errorf("Binary round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseVehicleRegistrationIdentificationRecordArray(t *testing.T) {
	tests := []struct {
		name       string
		recordType byte
		record     []byte
		wantNation ddv1.NationNumeric
	}{
		{
			name:       "canonical Gen2 V2 identification",
			recordType: recordTypeVehicleRegistrationIdentification,
			record:     []byte{0x15, 0x01, 'M', 'V', '2', '2', 'W', 'L', 'D', ' ', ' ', ' ', ' ', ' ', ' '},
			wantNation: ddv1.NationNumeric_UNITED_KINGDOM,
		},
		{
			name:       "Appendix 7 swapped number form",
			recordType: recordTypeVehicleRegistrationNumber,
			record:     []byte{0x01, 'M', 'V', '2', '2', 'W', 'L', 'D', ' ', ' ', ' ', ' ', ' ', ' '},
			wantNation: ddv1.NationNumeric_NATION_NUMERIC_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := appendRecordArrayHeader(nil, tt.recordType, uint16(len(tt.record)), 1)
			data = append(data, tt.record...)
			got, bytesRead, err := parseVehicleRegistrationIdentificationRecordArray(data, 0)
			if err != nil {
				t.Fatalf("parseVehicleRegistrationIdentificationRecordArray() error = %v", err)
			}
			if bytesRead != len(data) {
				t.Errorf("bytes read = %d, want %d", bytesRead, len(data))
			}
			if got.GetNation() != tt.wantNation {
				t.Errorf("nation = %v, want %v", got.GetNation(), tt.wantNation)
			}
			if got.GetNumber().GetValue() != "MV22WLD" {
				t.Errorf("registration = %q, want %q", got.GetNumber().GetValue(), "MV22WLD")
			}
		})
	}
}

func TestParseVehicleRegistrationIdentificationRecordArrayRejectsMismatchedTypeAndSize(t *testing.T) {
	data := appendRecordArrayHeader(nil, recordTypeVehicleRegistrationIdentification, lenVehicleRegistrationNumber, 1)
	data = append(data, make([]byte, lenVehicleRegistrationNumber)...)

	if _, _, err := parseVehicleRegistrationIdentificationRecordArray(data, 0); err == nil {
		t.Fatal("parseVehicleRegistrationIdentificationRecordArray() error = nil, want mismatch error")
	}
}
