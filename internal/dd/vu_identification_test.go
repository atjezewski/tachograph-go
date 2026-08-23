package dd

import (
	"bytes"
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// TestVuIdentification_Generation2Tail covers the components a generation 2
// vehicle unit appends to VuIdentification: the VU's own generation and its
// ability to use generation 1 cards, and — in version 2 — the digital map
// version. Stopping after the approval number leaves the record two bytes short
// on version 1 and fourteen short on version 2.
func TestVuIdentification_Generation2Tail(t *testing.T) {
	base := vuIdentificationGen2Base()

	for _, tt := range []struct {
		name                string
		data                []byte
		wantLen             int
		wantDigitalMapValue string
		wantDigitalMap      bool
	}{
		{
			name:    "version 1",
			data:    append(base, 0x02, 0x00),
			wantLen: 126,
		},
		{
			name: "version 2",
			data: append(append(base, 0x02, 0x00), []byte("MAP-2025    ")...),
			// IA5String(SIZE(12)), space padded.
			wantLen:             138,
			wantDigitalMapValue: "MAP-2025",
			wantDigitalMap:      true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) != tt.wantLen {
				t.Fatalf("test fixture is %d bytes, want %d", len(tt.data), tt.wantLen)
			}
			opts := UnmarshalOptions{PreserveRawData: true}
			ident, err := opts.UnmarshalVuIdentification(tt.data)
			if err != nil {
				t.Fatalf("unmarshal VuIdentification: %v", err)
			}
			if got := ident.GetVuGeneration(); got != ddv1.Generation_GENERATION_2 {
				t.Errorf("vu generation = %v, want GENERATION_2", got)
			}
			if !ident.GetSupportsGeneration_1Cards() {
				t.Error("supports generation 1 cards = false, want true: vuAbility bit 'a' is '0'B")
			}
			if got := ident.HasDigitalMapVersion(); got != tt.wantDigitalMap {
				t.Errorf("has digital map version = %v, want %v", got, tt.wantDigitalMap)
			}
			if tt.wantDigitalMap {
				if got := ident.GetDigitalMapVersion().GetValue(); got != tt.wantDigitalMapValue {
					t.Errorf("digital map version = %q, want %q", got, tt.wantDigitalMapValue)
				}
			}

			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalVuIdentification(ident)
			if err != nil {
				t.Fatalf("marshal VuIdentification: %v", err)
			}
			if !bytes.Equal(marshaled, tt.data) {
				t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, tt.data)
			}
		})
	}
}

func TestVuIdentification_UnsupportedGeneration1Cards(t *testing.T) {
	// vuAbility 'xxxxxxx1'B: the vehicle unit does not accept generation 1 cards.
	data := append(vuIdentificationGen2Base(), 0x02, 0x01)

	opts := UnmarshalOptions{PreserveRawData: true}
	ident, err := opts.UnmarshalVuIdentification(data)
	if err != nil {
		t.Fatalf("unmarshal VuIdentification: %v", err)
	}
	if ident.GetSupportsGeneration_1Cards() {
		t.Error("supports generation 1 cards = true, want false")
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalVuIdentification(ident)
	if err != nil {
		t.Fatalf("marshal VuIdentification: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

// vuIdentificationGen2Base builds the 124 bytes a generation 2 VuIdentification
// starts with: the components it shares with generation 1, with a 16-byte
// approval number.
func vuIdentificationGen2Base() []byte {
	data := make([]byte, 0, 124)
	name := make([]byte, 36)
	name[0] = 0x01 // ISO 8859-1
	copy(name[1:], []byte("MANUFACTURER                       "))
	data = append(data, name...)
	address := make([]byte, 36)
	address[0] = 0x01
	copy(address[1:], []byte("ADDRESS                            "))
	data = append(data, address...)
	data = append(data, []byte("PART-NUMBER-0001")...)                  // part number, IA5String(16)
	data = append(data, 0x00, 0x00, 0x00, 0x01, 0x25, 0x08, 0x01, 0xa1) // serial number
	data = append(data, []byte("1234")...)                              // software version
	data = append(data, 0x66, 0x21, 0x0a, 0xa4)                         // software installation date
	data = append(data, 0x66, 0x21, 0x0a, 0xa4)                         // manufacturing date
	data = append(data, []byte("APPROVAL-0000001")...)                  // approval number, IA5String(16)
	return data
}
