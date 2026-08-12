package dd

import (
	"bytes"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var expiryTimeReal = []byte{0x6e, 0x34, 0xe3, 0xff}

func TestVuCardIWRecordExpiryUsesTimeReal(t *testing.T) {
	data := make([]byte, 129)
	copy(data[90:94], expiryTimeReal)

	record, err := (UnmarshalOptions{}).UnmarshalVuCardIWRecord(data)
	if err != nil {
		t.Fatalf("UnmarshalVuCardIWRecord() error = %v", err)
	}
	assertExpiryTime(t, record.GetCardExpiryDate())

	encoded, err := (MarshalOptions{}).MarshalVuCardIWRecord(record)
	if err != nil {
		t.Fatalf("MarshalVuCardIWRecord() error = %v", err)
	}
	if !bytes.Equal(encoded[90:94], expiryTimeReal) {
		t.Fatalf("encoded expiry = %x, want %x", encoded[90:94], expiryTimeReal)
	}
}

func TestVuCardIWRecordG2ExpiryUsesTimeReal(t *testing.T) {
	data := make([]byte, 131)
	copy(data[91:95], expiryTimeReal)
	data[129] = 2 // PreviousVehicleInfo.vuGeneration

	record, err := (UnmarshalOptions{}).UnmarshalVuCardIWRecordG2(data)
	if err != nil {
		t.Fatalf("UnmarshalVuCardIWRecordG2() error = %v", err)
	}
	assertExpiryTime(t, record.GetCardExpiryDate())

	encoded, err := (MarshalOptions{}).MarshalVuCardIWRecordG2(record)
	if err != nil {
		t.Fatalf("MarshalVuCardIWRecordG2() error = %v", err)
	}
	if !bytes.Equal(encoded[91:95], expiryTimeReal) {
		t.Fatalf("encoded expiry = %x, want %x", encoded[91:95], expiryTimeReal)
	}
}

func TestVuCalibrationRecordExpiryUsesTimeReal(t *testing.T) {
	data := make([]byte, 167)
	copy(data[91:95], expiryTimeReal)

	record, err := (UnmarshalOptions{}).UnmarshalVuCalibrationRecord(data)
	if err != nil {
		t.Fatalf("UnmarshalVuCalibrationRecord() error = %v", err)
	}
	assertExpiryTime(t, record.GetWorkshopCardExpiryDate())

	encoded, err := (MarshalOptions{}).MarshalVuCalibrationRecord(record)
	if err != nil {
		t.Fatalf("MarshalVuCalibrationRecord() error = %v", err)
	}
	if !bytes.Equal(encoded[91:95], expiryTimeReal) {
		t.Fatalf("encoded expiry = %x, want %x", encoded[91:95], expiryTimeReal)
	}
}

func assertExpiryTime(t *testing.T, got *timestamppb.Timestamp) {
	t.Helper()
	if got == nil {
		t.Fatal("expiry time is nil")
	}
	want := time.Date(2028, 8, 3, 23, 59, 59, 0, time.UTC)
	if !got.AsTime().Equal(want) {
		t.Fatalf("expiry time = %s, want %s", got.AsTime(), want)
	}
}
