package card

import (
	"bytes"
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// The Gen2v2 elementary files below all share the same shape: a two-byte
// pointer to the newest record, followed by fixed-size record slots.

func TestPlacesAuthentication_RoundTrip(t *testing.T) {
	data := []byte{
		0x00, 0x01, // placeAuthPointerNewestRecord
		0x00, 0x00, 0x00, 0x0a, 0x01, // entryTime, authenticationStatus
		0x00, 0x00, 0x00, 0x00, 0x00, // unused slot
	}

	placesAuth, err := UnmarshalOptions{}.unmarshalPlacesAuthentication(data)
	if err != nil {
		t.Fatalf("unmarshal places authentication: %v", err)
	}
	if got := placesAuth.GetNewestRecordIndex(); got != 1 {
		t.Errorf("newest record index = %d, want 1", got)
	}
	if got := len(placesAuth.GetRecords()); got != 2 {
		t.Fatalf("got %d records, want 2", got)
	}
	record := placesAuth.GetRecords()[0]
	if got := record.GetEntryTime().GetSeconds(); got != 10 {
		t.Errorf("entry time = %d, want 10", got)
	}
	if got := record.GetAuthenticationStatus(); got != ddv1.PositionAuthenticationStatus_AUTHENTICATED {
		t.Errorf("authentication status = %v, want AUTHENTICATED", got)
	}

	marshaled, err := MarshalOptions{}.MarshalPlacesAuthentication(placesAuth)
	if err != nil {
		t.Fatalf("marshal places authentication: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

func TestGnssPlacesAuthentication_RoundTrip(t *testing.T) {
	data := []byte{
		0x00, 0x00, // gnssAuthADPointerNewestRecord
		0x00, 0x00, 0x00, 0x0a, 0x00, // timeStamp, authenticationStatus
	}

	gnssAuth, err := UnmarshalOptions{}.unmarshalGnssPlacesAuthentication(data)
	if err != nil {
		t.Fatalf("unmarshal GNSS places authentication: %v", err)
	}
	if got := len(gnssAuth.GetRecords()); got != 1 {
		t.Fatalf("got %d records, want 1", got)
	}
	if got := gnssAuth.GetRecords()[0].GetAuthenticationStatus(); got != ddv1.PositionAuthenticationStatus_NOT_AUTHENTICATED {
		t.Errorf("authentication status = %v, want NOT_AUTHENTICATED", got)
	}

	marshaled, err := MarshalOptions{}.MarshalGnssPlacesAuthentication(gnssAuth)
	if err != nil {
		t.Fatalf("marshal GNSS places authentication: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

func TestLoadTypeEntries_RoundTrip(t *testing.T) {
	data := []byte{
		0x00, 0x00, // loadTypeEntryPointerNewestRecord
		0x00, 0x00, 0x00, 0x0a, 0x01, // timeStamp, loadTypeEntered (goods)
		0x00, 0x00, 0x00, 0x00, 0x00, // unused slot: load type not defined
	}

	loadTypeEntries, err := UnmarshalOptions{}.unmarshalLoadTypeEntries(data)
	if err != nil {
		t.Fatalf("unmarshal load type entries: %v", err)
	}
	if got := len(loadTypeEntries.GetRecords()); got != 2 {
		t.Fatalf("got %d records, want 2", got)
	}
	if got := loadTypeEntries.GetRecords()[0].GetLoadTypeEntered(); got != ddv1.LoadType_GOODS {
		t.Errorf("load type = %v, want GOODS", got)
	}
	if got := loadTypeEntries.GetRecords()[1].GetLoadTypeEntered(); got != ddv1.LoadType_NOT_DEFINED {
		t.Errorf("unused slot load type = %v, want NOT_DEFINED", got)
	}

	marshaled, err := MarshalOptions{}.MarshalLoadTypeEntries(loadTypeEntries)
	if err != nil {
		t.Fatalf("marshal load type entries: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

func TestLoadUnloadOperations_UnusedSlotIsNotAnError(t *testing.T) {
	// An unused slot carries '00'H in operationType, which the regulation
	// reserves. Cards in service are full of them, so parsing must not fail.
	data := []byte{
		0x00, 0x00, // loadUnloadPointerNewestRecord
		// One load operation.
		0x00, 0x00, 0x00, 0x0a, // timeStamp
		0x01,                   // operationType: load
		0x00, 0x00, 0x00, 0x0a, // gnssPlaceAuthRecord: timeStamp
		0x03,             // gnssAccuracy
		0x00, 0x0e, 0xa6, // latitude
		0x00, 0x09, 0x94, // longitude
		0x01,             // authenticationStatus
		0x00, 0x00, 0x64, // vehicleOdometerValue
		// One unused slot.
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	loadUnload, err := UnmarshalOptions{}.unmarshalLoadUnloadOperations(data)
	if err != nil {
		t.Fatalf("unmarshal load/unload operations: %v", err)
	}
	if got := len(loadUnload.GetRecords()); got != 2 {
		t.Fatalf("got %d records, want 2", got)
	}
	if got := loadUnload.GetRecords()[0].GetOperationType(); got != ddv1.OperationType_LOAD_OPERATION {
		t.Errorf("operation type = %v, want LOAD_OPERATION", got)
	}
	unused := loadUnload.GetRecords()[1]
	if got := unused.GetOperationType(); got != ddv1.OperationType_OPERATION_TYPE_UNRECOGNIZED {
		t.Errorf("unused slot operation type = %v, want OPERATION_TYPE_UNRECOGNIZED", got)
	}
	if got := unused.GetUnrecognizedOperationType(); got != 0 {
		t.Errorf("unused slot raw operation type = %d, want 0", got)
	}

	marshaled, err := MarshalOptions{}.MarshalLoadUnloadOperations(loadUnload)
	if err != nil {
		t.Fatalf("marshal load/unload operations: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
}

// TestParseRawDriverCardFile_Gen2V2Files covers the five elementary files that
// second-generation version 2 driver cards carry and that the parser used to
// drop on the floor: they have proto fields and unmarshallers, but no case in
// the dispatch, and the switch has no default that would have noticed.
func TestParseRawDriverCardFile_Gen2V2Files(t *testing.T) {
	var data []byte
	appendEF := func(fid uint16, value []byte) {
		data = append(data, byte(fid>>8), byte(fid), 0x02, byte(len(value)>>8), byte(len(value)))
		data = append(data, value...)
		// Every one of these files is signed on a real card.
		data = append(data, byte(fid>>8), byte(fid), 0x03, 0x00, 0x40)
		data = append(data, bytes.Repeat([]byte{0xab}, 64)...)
	}
	appendEF(0x0526, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x01})
	appendEF(0x0527, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00})
	appendEF(0x0528, []byte{
		0x00, 0x00,
		0x0f, 0x11,
		0x00, 0x00, 0x00, 0x0a, 0x03, 0x00, 0x0e, 0xa6, 0x00, 0x09, 0x94, 0x01,
		0x00, 0x00, 0x64,
	})
	appendEF(0x0529, []byte{
		0x00, 0x00,
		0x00, 0x00, 0x00, 0x0a, 0x01,
		0x00, 0x00, 0x00, 0x0a, 0x03, 0x00, 0x0e, 0xa6, 0x00, 0x09, 0x94, 0x01,
		0x00, 0x00, 0x64,
	})
	appendEF(0x0530, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x01})

	rawFile, err := UnmarshalOptions{}.UnmarshalRawCardFile(data)
	if err != nil {
		t.Fatalf("unmarshal raw card file: %v", err)
	}
	file, err := ParseOptions{PreserveRawData: true}.ParseRawDriverCardFile(rawFile)
	if err != nil {
		t.Fatalf("parse raw driver card file: %v", err)
	}

	g2 := file.GetTachographG2()
	if g2 == nil {
		t.Fatal("no Gen2 application on the parsed card")
	}
	for _, tt := range []struct {
		name    string
		records int
		message interface{ GetSignature() []byte }
	}{
		{"places authentication", len(g2.GetPlacesAuthentication().GetRecords()), g2.GetPlacesAuthentication()},
		{"GNSS places authentication", len(g2.GetGnssPlacesAuthentication().GetRecords()), g2.GetGnssPlacesAuthentication()},
		{"border crossings", len(g2.GetBorderCrossings().GetRecords()), g2.GetBorderCrossings()},
		{"load/unload operations", len(g2.GetLoadUnloadOperations().GetRecords()), g2.GetLoadUnloadOperations()},
		{"load type entries", len(g2.GetLoadTypeEntries().GetRecords()), g2.GetLoadTypeEntries()},
	} {
		if tt.message == nil {
			t.Errorf("%s did not reach the parsed model", tt.name)
			continue
		}
		if tt.records != 1 {
			t.Errorf("%s has %d records, want 1", tt.name, tt.records)
		}
		if len(tt.message.GetSignature()) != 64 {
			t.Errorf("%s signature is %d bytes, want 64", tt.name, len(tt.message.GetSignature()))
		}
	}

	if got := g2.GetBorderCrossings().GetRecords()[0].GetCountryEntered(); got != ddv1.NationNumeric_FRANCE {
		t.Errorf("border crossing country entered = %v, want FRANCE", got)
	}
	if got := g2.GetLoadUnloadOperations().GetRecords()[0].GetOperationType(); got != ddv1.OperationType_LOAD_OPERATION {
		t.Errorf("load/unload operation type = %v, want LOAD_OPERATION", got)
	}

	// Both serializers must put the files back on the card.
	marshaled, err := MarshalOptions{}.MarshalDriverCardFile(file)
	if err != nil {
		t.Fatalf("marshal driver card file: %v", err)
	}
	if !bytes.Equal(marshaled, data) {
		t.Errorf("MarshalDriverCardFile round trip mismatch:\n got %x\nwant %x", marshaled, data)
	}
	unparsed, err := UnparseDriverCardFile(file)
	if err != nil {
		t.Fatalf("unparse driver card file: %v", err)
	}
	remarshaled, err := MarshalOptions{}.MarshalRawCardFile(unparsed)
	if err != nil {
		t.Fatalf("marshal raw card file: %v", err)
	}
	if !bytes.Equal(remarshaled, data) {
		t.Errorf("Unparse round trip mismatch:\n got %x\nwant %x", remarshaled, data)
	}
}
