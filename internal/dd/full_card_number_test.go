package dd

import (
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// TestUnmarshalFullCardNumber_IssuingState guards against the same enum-mapping
// regression as the vehicle registration nation: the issuing member state byte
// must be mapped through the protocol_enum_value annotation, not cast directly
// to the proto enum. With the raw cast, a Norwegian-issued card (protocol value
// 37) decoded as MONACO (proto enum value 37). See tacho#1090.
func TestUnmarshalFullCardNumber_IssuingState(t *testing.T) {
	// 16-byte DriverIdentification: 14-byte IA5 number + replacement + renewal.
	driverID := []byte{
		0x44, 0x52, 0x49, 0x56, 0x45, 0x52, 0x49, 0x44, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31, // "DRIVERID000001"
		0x31, // replacement index "1"
		0x31, // renewal index "1"
	}

	tests := []struct {
		name        string
		issuingByte byte
		wantNation  ddv1.NationNumeric
	}{
		{name: "Norway issuing state", issuingByte: 0x25, wantNation: ddv1.NationNumeric_NORWAY},
		{name: "Monaco issuing state", issuingByte: 0x22, wantNation: ddv1.NationNumeric_MONACO},
		{name: "Unrecognized issuing state", issuingByte: 0x64, wantNation: ddv1.NationNumeric_NATION_NUMERIC_UNRECOGNIZED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := append([]byte{0x01, tt.issuingByte}, driverID...) // 0x01 = DRIVER_CARD protocol value

			unmarshalOpts := UnmarshalOptions{PreserveRawData: true}
			got, err := unmarshalOpts.UnmarshalFullCardNumber(input)
			if err != nil {
				t.Fatalf("UnmarshalFullCardNumber() unexpected error: %v", err)
			}
			if got.GetCardIssuingMemberState() != tt.wantNation {
				t.Errorf("GetCardIssuingMemberState() = %v, want %v", got.GetCardIssuingMemberState(), tt.wantNation)
			}

			// Round-trip: the issuing-state byte must be preserved.
			var marshalOpts MarshalOptions
			out, err := marshalOpts.MarshalFullCardNumber(got)
			if err != nil {
				t.Fatalf("MarshalFullCardNumber() unexpected error: %v", err)
			}
			if out[1] != tt.issuingByte {
				t.Errorf("round-trip issuing-state byte = 0x%02X, want 0x%02X", out[1], tt.issuingByte)
			}
		})
	}
}

func TestFullCardNumberControlCard(t *testing.T) {
	input := []byte{
		0x03, 0x0B, // control card, United Kingdom
		'C', 'O', 'N', 'T', 'R', 'O', 'L', '0', '0', '0', '0', '0', '1',
		'0', '0', '0',
	}

	opts := UnmarshalOptions{PreserveRawData: true}
	got, err := opts.UnmarshalFullCardNumber(input)
	if err != nil {
		t.Fatalf("UnmarshalFullCardNumber() error = %v", err)
	}
	if got.GetCardType() != ddv1.EquipmentType_CONTROL_CARD {
		t.Fatalf("card type = %v, want CONTROL_CARD", got.GetCardType())
	}
	if got.GetOwnerIdentification() == nil {
		t.Fatal("owner identification is nil")
	}
	if got.GetOwnerIdentification().GetOwnerIdentification().GetValue() != "CONTROL000001" {
		t.Errorf(
			"owner identification = %q, want %q",
			got.GetOwnerIdentification().GetOwnerIdentification().GetValue(),
			"CONTROL000001",
		)
	}

	marshaled, err := (MarshalOptions{}).MarshalFullCardNumber(got)
	if err != nil {
		t.Fatalf("MarshalFullCardNumber() error = %v", err)
	}
	if string(marshaled) != string(input) {
		t.Errorf("round trip = %x, want %x", marshaled, input)
	}
}
