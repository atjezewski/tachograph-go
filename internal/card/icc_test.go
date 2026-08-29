package card

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

func TestICC_Generation1(t *testing.T) {
	// Discover all matching hexdump files using type-safe enums
	hexdumpFiles, err := findHexdumpFiles(
		cardv1.ElementaryFileType_EF_ICC,
		ddv1.Generation_GENERATION_1,
		cardv1.ContentType_DATA,
	)
	if err != nil {
		t.Fatalf("Failed to discover hexdump files: %v", err)
	}
	if len(hexdumpFiles) == 0 {
		t.Fatal("No hexdump files found for EF_ICC GENERATION_1")
	}

	// Run subtest for each discovered file
	for _, hexdumpPath := range hexdumpFiles {
		// Use relative path from testdata as subtest name
		relPath := strings.TrimPrefix(hexdumpPath, "testdata/records/")
		testName := strings.TrimSuffix(relPath, ".hexdump")

		t.Run(testName, func(t *testing.T) {
			// Read hexdump
			data, err := readHexdump(hexdumpPath)
			if err != nil {
				t.Fatalf("Failed to read hexdump: %v", err)
			}

			// Unmarshal
			opts := UnmarshalOptions{}
			icc, err := opts.unmarshalIcc(data)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Golden JSON comparison
			goldenPath := goldenJSONPath(hexdumpPath)
			loadOrCreateGolden(t, icc, goldenPath)

			// Round-trip test
			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalIcc(icc)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if diff := cmp.Diff(data, marshaled); diff != "" {
				t.Errorf("Binary round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestICC_ModuleEmbedderKeepsBinaryBytes covers the module embedder code in
// EF_ICC. Cards carry a binary code there, not text: reading it as an IA5
// string drops any byte that is not printable.
func TestICC_ModuleEmbedderKeepsBinaryBytes(t *testing.T) {
	data := []byte{
		0x00,                                           // clockStop
		0x00, 0xbc, 0x61, 0x4e, 0x01, 0x20, 0x01, 0x99, // cardExtendedSerialNumber
		'A', 'P', 'P', 'R', '0', '0', '0', '1', // cardApprovalNumber
		0xaa,     // cardPersonaliserId
		'F', 'R', // embedderIcAssemblerId: countryCode
		0x01, 0x63, // embedderIcAssemblerId: moduleEmbedder, a binary code
		0xa3,       // embedderIcAssemblerId: manufacturerInformation
		0xcc, 0xdd, // icIdentifier
	}

	opts := UnmarshalOptions{}
	icc, err := opts.unmarshalIcc(data)
	if err != nil {
		t.Fatalf("unmarshal ICC: %v", err)
	}
	if got, want := icc.GetEmbedderIcAssemblerId().GetModuleEmbedderRaw(), []byte{0x01, 0x63}; !bytes.Equal(got, want) {
		t.Errorf("module embedder = %x, want %x", got, want)
	}
	if got := icc.GetEmbedderIcAssemblerId().GetModuleEmbedder(); got == nil {
		t.Error("compatibility module embedder view was not populated")
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalIcc(icc)
	if err != nil {
		t.Fatalf("marshal ICC: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}

	legacy, err := opts.UnmarshalIa5StringValue([]byte("AB"))
	if err != nil {
		t.Fatalf("unmarshal legacy module embedder: %v", err)
	}
	legacyEIA := &cardv1.Icc_EmbedderIcAssemblerId{}
	legacyEIA.SetCountryCode(legacy)
	legacyEIA.SetModuleEmbedder(legacy)
	marshaledEIA, err := marshalOpts.MarshalEmbedderIcAssemblerId(legacyEIA)
	if err != nil {
		t.Fatalf("marshal legacy module embedder: %v", err)
	}
	if got, want := marshaledEIA[2:4], []byte("AB"); !bytes.Equal(got, want) {
		t.Errorf("legacy module embedder = %x, want %x", got, want)
	}

	legacyEIA.SetModuleEmbedderRaw([]byte{0x01})
	if _, err := marshalOpts.MarshalEmbedderIcAssemblerId(legacyEIA); err == nil {
		t.Error("expected invalid raw module embedder length to be rejected")
	}
}
