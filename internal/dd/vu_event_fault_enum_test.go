package dd

import (
	"bytes"
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

func TestEventFaultEnumsUseProtocolValues(t *testing.T) {
	unmarshalOpts := UnmarshalOptions{}
	marshalOpts := MarshalOptions{}

	for _, tt := range []struct {
		name       string
		raw        byte
		want       ddv1.EventFaultType
		unknownRaw int32
	}{
		{"no further details", 0x00, ddv1.EventFaultType_GENERAL_NO_FURTHER_DETAILS, 0},
		{"overspeeding", 0x07, ddv1.EventFaultType_GENERAL_OVER_SPEEDING, 0},
		{"unrecognized", 0xff, ddv1.EventFaultType_EVENT_FAULT_TYPE_UNSPECIFIED, 0xff},
	} {
		t.Run("type/"+tt.name, func(t *testing.T) {
			got, unknownRaw := unmarshalOpts.parseEventFaultType(tt.raw)
			if got != tt.want || unknownRaw != tt.unknownRaw {
				t.Fatalf("parse 0x%02x = (%v, 0x%02x), want (%v, 0x%02x)", tt.raw, got, unknownRaw, tt.want, tt.unknownRaw)
			}
			marshaled, err := marshalOpts.marshalEventFaultType(got, unknownRaw)
			if err != nil {
				t.Fatalf("marshal event/fault type: %v", err)
			}
			if marshaled != tt.raw {
				t.Errorf("marshal = 0x%02x, want 0x%02x", marshaled, tt.raw)
			}
		})
	}

	for _, tt := range []struct {
		name       string
		raw        byte
		want       ddv1.EventFaultRecordPurpose
		unknownRaw int32
	}{
		{"ten most recent", 0x00, ddv1.EventFaultRecordPurpose_TEN_MOST_RECENT, 0},
		{"active or ongoing", 0x07, ddv1.EventFaultRecordPurpose_ACTIVE_OR_ONGOING, 0},
		{"unrecognized", 0xff, ddv1.EventFaultRecordPurpose_EVENT_FAULT_RECORD_PURPOSE_UNSPECIFIED, 0xff},
	} {
		t.Run("purpose/"+tt.name, func(t *testing.T) {
			got, unknownRaw := unmarshalOpts.parseEventFaultRecordPurpose(tt.raw)
			if got != tt.want || unknownRaw != tt.unknownRaw {
				t.Fatalf("parse 0x%02x = (%v, 0x%02x), want (%v, 0x%02x)", tt.raw, got, unknownRaw, tt.want, tt.unknownRaw)
			}
			marshaled, err := marshalOpts.marshalEventFaultRecordPurpose(got, unknownRaw)
			if err != nil {
				t.Fatalf("marshal event/fault purpose: %v", err)
			}
			if marshaled != tt.raw {
				t.Errorf("marshal = 0x%02x, want 0x%02x", marshaled, tt.raw)
			}
		})
	}
}

func TestVuOverspeedEventRecordEnumRoundTrip(t *testing.T) {
	data := make([]byte, 31)
	data[0] = 0x07 // GENERAL_OVER_SPEEDING
	data[1] = 0x00 // TEN_MOST_RECENT
	for i := 12; i < 30; i++ {
		data[i] = 0xff // no card inserted
	}

	record, err := (UnmarshalOptions{PreserveRawData: true}).UnmarshalVuOverspeedEventRecord(data)
	if err != nil {
		t.Fatalf("unmarshal VuOverspeedEventRecord: %v", err)
	}
	if got := record.GetEventType(); got != ddv1.EventFaultType_GENERAL_OVER_SPEEDING {
		t.Errorf("event type = %v, want GENERAL_OVER_SPEEDING", got)
	}
	if got := record.GetRecordPurpose(); got != ddv1.EventFaultRecordPurpose_TEN_MOST_RECENT {
		t.Errorf("record purpose = %v, want TEN_MOST_RECENT", got)
	}

	marshaled, err := (MarshalOptions{}).MarshalVuOverspeedEventRecord(record)
	if err != nil {
		t.Fatalf("marshal VuOverspeedEventRecord: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}
