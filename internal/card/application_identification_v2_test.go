package card

import (
	"bytes"
	"testing"

	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
)

// TestApplicationIdentificationV2_RoundTrip covers the ten-byte
// EF_Application_Identification_V2 record of a driver card.
//
// Every component is a two-byte INTEGER(0..2^16-1). Read as four one-byte
// counts the record yields four wrong numbers and six unread bytes, and the
// numbers matter: they are the sizes of the ring buffers that the rest of the
// Gen2v2 application is stored in.
func TestApplicationIdentificationV2_RoundTrip(t *testing.T) {
	data := []byte{
		0x00, 0x08, // lengthOfFollowingData
		0x04, 0x60, // noOfBorderCrossingRecords: 1120
		0x06, 0x58, // noOfLoadUnloadRecords: 1624
		0x01, 0x50, // noOfLoadTypeEntryRecords: 336
		0x0c, 0x00, // vuConfigurationLengthRange: 3072
	}

	unmarshalOpts := UnmarshalOptions{}
	appIdV2, err := unmarshalOpts.unmarshalApplicationIdentificationV2(data)
	if err != nil {
		t.Fatalf("unmarshal application identification V2: %v", err)
	}
	if got := appIdV2.GetCardType(); got != cardv1.CardType_DRIVER_CARD {
		t.Errorf("card type = %v, want DRIVER_CARD", got)
	}
	driver := appIdV2.GetDriver()
	for _, tt := range []struct {
		name string
		got  int32
		want int32
	}{
		{"lengthOfFollowingData", driver.GetLengthOfFollowingData(), 8},
		{"noOfBorderCrossingRecords", driver.GetBorderCrossingRecordsCount(), 1120},
		{"noOfLoadUnloadRecords", driver.GetLoadUnloadRecordsCount(), 1624},
		{"noOfLoadTypeEntryRecords", driver.GetLoadTypeEntryRecordsCount(), 336},
		{"vuConfigurationLengthRange", driver.GetVuConfigurationLengthRange(), 3072},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalCardApplicationIdentificationV2(appIdV2)
	if err != nil {
		t.Fatalf("marshal application identification V2: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

func TestApplicationIdentificationV2_RejectsShortRecord(t *testing.T) {
	// The four one-byte reading of this record consumed exactly this much.
	opts := UnmarshalOptions{}
	if _, err := opts.unmarshalApplicationIdentificationV2([]byte{0x00, 0x08, 0x04, 0x60}); err == nil {
		t.Error("unmarshal accepted a four-byte record, want an error")
	}
}
