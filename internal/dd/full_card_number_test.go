package dd

import (
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

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
