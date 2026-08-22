package vu

import (
	"bytes"
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

func TestParseVuPlaceDailyWorkPeriodRecordArrayG2PreservesCard(t *testing.T) {
	recordData := testVuPlaceRecordData(40)
	data := appendRecordArrayHeader(nil, 0x05, 40, 1)
	data = append(data, recordData...)

	records, consumed, err := parseVuPlaceDailyWorkPeriodRecordArrayG2(data, 0)
	if err != nil {
		t.Fatalf("parse place records: %v", err)
	}
	if consumed != len(data) {
		t.Fatalf("consumed %d bytes, want %d", consumed, len(data))
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	card := records[0].GetFullCardNumber()
	if got := card.GetGeneration(); got != ddv1.Generation_GENERATION_2 {
		t.Errorf("generation = %v, want GENERATION_2", got)
	}
	if got := card.GetFullCardNumber().GetRawData(); !bytes.Equal(got, recordData[:18]) {
		t.Errorf("card raw data = %x, want %x", got, recordData[:18])
	}

	marshaled, err := marshalPlaceRecordsG2V1(records)
	if err != nil {
		t.Fatalf("marshal place records: %v", err)
	}
	if !bytes.Equal(marshaled, recordData) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, recordData)
	}
}

func TestParseVuPlaceDailyWorkPeriodRecordArrayG2V2PreservesCardAndAuthentication(t *testing.T) {
	recordData := testVuPlaceRecordData(41)
	recordData[40] = 0x01
	data := appendRecordArrayHeader(nil, 0x05, 41, 1)
	data = append(data, recordData...)

	records, consumed, err := parseVuPlaceDailyWorkPeriodRecordArrayG2V2(data, 0)
	if err != nil {
		t.Fatalf("parse place records: %v", err)
	}
	if consumed != len(data) {
		t.Fatalf("consumed %d bytes, want %d", consumed, len(data))
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	card := records[0].GetFullCardNumber()
	if got := card.GetGeneration(); got != ddv1.Generation_GENERATION_2 {
		t.Errorf("generation = %v, want GENERATION_2", got)
	}
	if got := card.GetFullCardNumber().GetRawData(); !bytes.Equal(got, recordData[:18]) {
		t.Errorf("card raw data = %x, want %x", got, recordData[:18])
	}
	if got := records[0].GetPlaceAuthRecord().GetEntryGnssPlaceAuthRecord().GetAuthenticationStatus(); got != ddv1.PositionAuthenticationStatus_AUTHENTICATED {
		t.Errorf("authentication status = %v, want AUTHENTICATED", got)
	}

	marshaled, err := marshalPlaceRecordsG2V2(records)
	if err != nil {
		t.Fatalf("marshal place records: %v", err)
	}
	if !bytes.Equal(marshaled, recordData) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, recordData)
	}
}

func testVuPlaceRecordData(size int) []byte {
	data := make([]byte, size)
	data[0] = 0x01 // Driver card.
	data[1] = 0x15 // United Kingdom.
	copy(data[2:16], []byte("ABCDEFGHIJKLMN"))
	data[16] = '0'
	data[17] = '0'
	data[18] = 0x02 // Generation 2.
	data[23] = 0x00 // Begin daily work period.
	data[24] = 0x15 // United Kingdom.
	data[25] = 0x07 // Country-specific region code.
	return data
}
