package vu

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
)

func TestTechnicalData_Gen2V2(t *testing.T) {
	// Discover all matching hexdump files
	hexdumpFiles, err := findHexdumpFiles(vuv1.TransferType_TECHNICAL_DATA_GEN2_V2)
	if err != nil {
		t.Fatalf("Failed to discover hexdump files: %v", err)
	}
	if len(hexdumpFiles) == 0 {
		t.Skip("No hexdump files found for TECHNICAL_DATA_GEN2_V2")
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
			technicalData, err := unmarshalTechnicalDataGen2V2(data)
			if err != nil {
				t.Fatalf("Failed to unmarshal TechnicalData Gen2V2: %v", err)
			}
			if technicalData == nil {
				t.Fatal("Unmarshal returned nil")
			}

			// Golden JSON comparison
			goldenPath := goldenJSONPath(hexdumpPath)
			loadOrCreateGolden(t, technicalData, goldenPath)

			// Round-trip test - marshal
			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalTechnicalDataGen2V2(technicalData)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if diff := cmp.Diff(data, marshaled); diff != "" {
				t.Errorf("Binary round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTechnicalDataGen2V2IncludesAllRecordArrays(t *testing.T) {
	data, err := readHexdump("testdata/records/003-anonymized/099-TECHNICAL_DATA_GEN2_V2.hexdump")
	if err != nil {
		t.Fatal(err)
	}

	technicalData, err := unmarshalTechnicalDataGen2V2(data)
	if err != nil {
		t.Fatalf("unmarshalTechnicalDataGen2V2() error = %v", err)
	}
	if got := len(technicalData.GetCardRecords()); got != 33 {
		t.Errorf("card records = %d, want 33", got)
	}
	if got := len(technicalData.GetItsConsentRecords()); got != 28 {
		t.Errorf("ITS consent records = %d, want 28", got)
	}
	if got := len(technicalData.GetPowerSupplyInterruptions()); got != 3 {
		t.Errorf("power supply interruption records = %d, want 3", got)
	}
	if signature := technicalData.GetSignature(); len(signature) != 5 || signature[0] != recordTypeSignature {
		t.Errorf("signature = %x, want empty SignatureRecordArray", signature)
	}
}

func TestSizeOfTechnicalDataGen2V2IncludesCardArray(t *testing.T) {
	var data []byte
	for _, recordType := range []byte{
		recordTypeVuIdentification,
		recordTypeSensorPairedRecord,
		recordTypeSensorExternalGNSSCoupled,
		recordTypeVuCalibrationRecord,
		recordTypeVuCardRecord,
		recordTypeVuITSConsentRecord,
		recordTypeVuPowerSupplyInterruption,
	} {
		data = appendRecordArrayHeader(data, recordType, 1, 0)
	}
	data = appendRecordArrayHeader(data, recordTypeSignature, 64, 1)
	data = append(data, make([]byte, 64)...)

	totalSize, signatureSize, err := sizeOfTechnicalDataGen2V2(data)
	if err != nil {
		t.Fatalf("sizeOfTechnicalDataGen2V2() error = %v", err)
	}
	if totalSize != len(data) {
		t.Errorf("total size = %d, want %d", totalSize, len(data))
	}
	if signatureSize != 69 {
		t.Errorf("signature size = %d, want 69", signatureSize)
	}
}
