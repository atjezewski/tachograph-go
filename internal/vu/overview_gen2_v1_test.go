package vu

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
)

func TestOverview_Gen2V1(t *testing.T) {
	// Discover all matching hexdump files
	hexdumpFiles, err := findHexdumpFiles(vuv1.TransferType_OVERVIEW_GEN2_V1)
	if err != nil {
		t.Fatalf("Failed to discover hexdump files: %v", err)
	}
	if len(hexdumpFiles) == 0 {
		t.Skip("No hexdump files found for OVERVIEW_GEN2_V1")
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
			overview, err := unmarshalOverviewGen2V1(data)
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
			marshaled, err := marshalOpts.MarshalOverviewGen2V1(overview)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if diff := cmp.Diff(data, marshaled); diff != "" {
				t.Errorf("Binary round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseVehicleRegistrationNumberRecordArray(t *testing.T) {
	data := []byte{
		0x0B, 0x00, 0x0E, 0x00, 0x01, // RecordArray: VRN, 14 bytes, one record
		0x00, 'P', 'E', '6', '7', 'H', 'X', 'S', ' ', ' ', ' ', ' ', ' ', ' ',
	}

	got, bytesRead, err := parseVehicleRegistrationNumberRecordArray(data, 0)
	if err != nil {
		t.Fatalf("parseVehicleRegistrationNumberRecordArray() error = %v", err)
	}
	if bytesRead != len(data) {
		t.Errorf("bytes read = %d, want %d", bytesRead, len(data))
	}
	if got.GetNation() != ddv1.NationNumeric_NATION_NUMERIC_UNSPECIFIED {
		t.Errorf("nation = %v, want NATION_NUMERIC_UNSPECIFIED", got.GetNation())
	}
	if got.GetNumber().GetValue() != "PE67HXS" {
		t.Errorf("registration = %q, want %q", got.GetNumber().GetValue(), "PE67HXS")
	}
}

func TestParseVehicleRegistrationNumberRecordArrayAcceptsAppendix7SwappedForm(t *testing.T) {
	record := []byte{0x15, 0x01, 'P', 'E', '6', '7', 'H', 'X', 'S', ' ', ' ', ' ', ' ', ' ', ' '}
	data := appendRecordArrayHeader(nil, recordTypeVehicleRegistrationIdentification, uint16(len(record)), 1)
	data = append(data, record...)

	got, _, err := parseVehicleRegistrationNumberRecordArray(data, 0)
	if err != nil {
		t.Fatalf("parseVehicleRegistrationNumberRecordArray() error = %v", err)
	}
	if got.GetNation() != ddv1.NationNumeric_UNITED_KINGDOM {
		t.Errorf("nation = %v, want UNITED_KINGDOM", got.GetNation())
	}
	if got.GetNumber().GetValue() != "PE67HXS" {
		t.Errorf("registration = %q, want %q", got.GetNumber().GetValue(), "PE67HXS")
	}
}

func TestMarshalOverviewGen2V1UsesVehicleRegistrationNumber(t *testing.T) {
	number := &ddv1.StringValue{}
	number.SetEncoding(ddv1.Encoding_ISO_8859_1)
	number.SetLength(13)
	number.SetValue("PE67HXS")
	registration := &ddv1.VehicleRegistrationIdentification{}
	registration.SetNation(ddv1.NationNumeric_UNITED_KINGDOM)
	registration.SetNumber(number)

	overview := &vuv1.OverviewGen2V1{}
	overview.SetVehicleRegistrationWithNation(registration)

	got, err := (MarshalOptions{}).MarshalOverviewGen2V1(overview)
	if err != nil {
		t.Fatalf("MarshalOverviewGen2V1() error = %v", err)
	}

	// Two empty certificate arrays and one empty VIN array precede the VRN.
	const vrnHeaderOffset = 15
	if got[vrnHeaderOffset+1] != 0 || got[vrnHeaderOffset+2] != 14 {
		t.Fatalf("VRN RecordArray size bytes = %x, want 000e", got[vrnHeaderOffset+1:vrnHeaderOffset+3])
	}
	vrn := got[vrnHeaderOffset+5 : vrnHeaderOffset+5+14]
	if string(vrn[1:8]) != "PE67HXS" {
		t.Errorf("VRN bytes = %x, want encoded PE67HXS", vrn)
	}
}
