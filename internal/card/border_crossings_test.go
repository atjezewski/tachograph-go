package card

import (
	"bytes"
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

func TestBorderCrossings_RoundTrip(t *testing.T) {
	// Two record slots: one crossing, and one unused slot as issued cards carry
	// them. The pointer to the newest record is two bytes wide.
	data := []byte{
		0x00, 0x01, // borderCrossingPointerNewestRecord
		// CardBorderCrossingRecord: Spain -> France, authenticated, 100 km
		0x0f,                   // countryLeft
		0x11,                   // countryEntered
		0x00, 0x00, 0x00, 0x0a, // gnssPlaceAuthRecord: timeStamp
		0x03,             // gnssPlaceAuthRecord: gnssAccuracy
		0x00, 0x0e, 0xa6, // gnssPlaceAuthRecord: latitude
		0x00, 0x09, 0x94, // gnssPlaceAuthRecord: longitude
		0x01,             // gnssPlaceAuthRecord: authenticationStatus
		0x00, 0x00, 0x64, // vehicleOdometerValue
		// Unused slot.
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	opts := UnmarshalOptions{}
	borderCrossings, err := opts.unmarshalBorderCrossings(data)
	if err != nil {
		t.Fatalf("unmarshal border crossings: %v", err)
	}
	if got := borderCrossings.GetNewestRecordIndex(); got != 1 {
		t.Errorf("newest record index = %d, want 1", got)
	}
	if got := len(borderCrossings.GetRecords()); got != 2 {
		t.Fatalf("got %d records, want 2: unused slots are kept so the file keeps its length", got)
	}

	crossing := borderCrossings.GetRecords()[0]
	if got := crossing.GetCountryLeft(); got != ddv1.NationNumeric_SPAIN {
		t.Errorf("country left = %v, want SPAIN", got)
	}
	if got := crossing.GetCountryEntered(); got != ddv1.NationNumeric_FRANCE {
		t.Errorf("country entered = %v, want FRANCE", got)
	}
	if got := crossing.GetVehicleOdometerKm(); got != 100 {
		t.Errorf("odometer = %d, want 100", got)
	}
	gnss := crossing.GetGnssPlaceAuthRecord()
	if got := gnss.GetTimestamp().GetSeconds(); got != 10 {
		t.Errorf("timestamp = %d, want 10", got)
	}
	if got := gnss.GetAuthenticationStatus(); got != ddv1.PositionAuthenticationStatus_AUTHENTICATED {
		t.Errorf("authentication status = %v, want AUTHENTICATED", got)
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalBorderCrossings(borderCrossings)
	if err != nil {
		t.Fatalf("marshal border crossings: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

func TestBorderCrossings_UnrecognizedCountryIsPreserved(t *testing.T) {
	// NationNumeric leaves '39'H..'FC'H reserved. A card carrying one must
	// still parse, and must marshal back to the same byte.
	data := []byte{
		0x00, 0x00,
		0x7f,                   // countryLeft, reserved for future use
		0x11,                   // countryEntered
		0x00, 0x00, 0x00, 0x0a, // gnssPlaceAuthRecord
		0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00,
		0x00, 0x00, 0x64,
	}

	opts := UnmarshalOptions{}
	borderCrossings, err := opts.unmarshalBorderCrossings(data)
	if err != nil {
		t.Fatalf("unmarshal border crossings: %v", err)
	}
	record := borderCrossings.GetRecords()[0]
	if got := record.GetCountryLeft(); got != ddv1.NationNumeric_NATION_NUMERIC_UNRECOGNIZED {
		t.Errorf("country left = %v, want NATION_NUMERIC_UNRECOGNIZED", got)
	}
	if got := record.GetUnrecognizedCountryLeft(); got != 0x7f {
		t.Errorf("unrecognized country left = %#x, want 0x7f", got)
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalBorderCrossings(borderCrossings)
	if err != nil {
		t.Fatalf("marshal border crossings: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}
