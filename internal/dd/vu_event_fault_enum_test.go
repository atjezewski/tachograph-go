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

// TestVuCalibrationRecordPurposeUsesProtocolValues covers the calibration
// purpose byte, which shares the defect the tests above cover for events and
// faults: the protobuf enum reserves numbers 0 and 1, so casting the protocol
// byte to an enum number shifts every purpose by two and reports a periodic
// inspection as a first installation.
//
// See Data Dictionary, Section 2.8, `CalibrationPurpose`.
func TestVuCalibrationRecordPurposeUsesProtocolValues(t *testing.T) {
	for _, tt := range []struct {
		name         string
		protocolByte byte
		want         ddv1.CalibrationPurpose
		unrecognized int32
	}{
		{name: "reserved", protocolByte: 0x00, want: ddv1.CalibrationPurpose_CALIBRATION_PURPOSE_RESERVED},
		{name: "activation", protocolByte: 0x01, want: ddv1.CalibrationPurpose_ACTIVATION},
		{name: "first installation", protocolByte: 0x02, want: ddv1.CalibrationPurpose_FIRST_INSTALLATION},
		{name: "installation", protocolByte: 0x03, want: ddv1.CalibrationPurpose_INSTALLATION},
		{name: "periodic inspection", protocolByte: 0x04, want: ddv1.CalibrationPurpose_PERIODIC_INSPECTION},
		{name: "VRN entry by company", protocolByte: 0x05, want: ddv1.CalibrationPurpose_VRN_ENTRY_BY_COMPANY},
		{name: "time adjustment", protocolByte: 0x06, want: ddv1.CalibrationPurpose_TIME_ADJUSTMENT},
		{
			// '07'H..'7F'H are reserved for future use.
			name:         "reserved for future use",
			protocolByte: 0x40,
			want:         ddv1.CalibrationPurpose_CALIBRATION_PURPOSE_UNRECOGNIZED,
			unrecognized: 0x40,
		},
		{
			// '80'H..'FF'H are manufacturer specific.
			name:         "manufacturer specific",
			protocolByte: 0x9a,
			want:         ddv1.CalibrationPurpose_CALIBRATION_PURPOSE_UNRECOGNIZED,
			unrecognized: 0x9a,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 167)
			data[0] = tt.protocolByte

			opts := UnmarshalOptions{PreserveRawData: true}
			record, err := opts.UnmarshalVuCalibrationRecord(data)
			if err != nil {
				t.Fatalf("unmarshal calibration record: %v", err)
			}
			if got := record.GetPurpose(); got != tt.want {
				t.Errorf("purpose for protocol byte %#02x = %v, want %v", tt.protocolByte, got, tt.want)
			}
			if got := record.GetUnrecognizedPurpose(); got != tt.unrecognized {
				t.Errorf("unrecognized purpose = %#02x, want %#02x", got, tt.unrecognized)
			}

			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalVuCalibrationRecord(record)
			if err != nil {
				t.Fatalf("marshal calibration record: %v", err)
			}
			if marshaled[0] != tt.protocolByte {
				t.Errorf("marshalled purpose byte = %#02x, want %#02x", marshaled[0], tt.protocolByte)
			}
		})
	}
}
