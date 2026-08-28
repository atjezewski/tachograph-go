package card

import (
	"encoding/binary"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// unmarshalPlacesAuthentication parses the EF_Places_Authentication file, which
// records whether the position stored for each entry in EF_Places was
// authenticated by the GNSS receiver.
//
// The data type `CardPlaceAuthDailyWorkPeriod` is specified in the Data Dictionary, Section 2.26a.
//
// ASN.1 Definition:
//
//	CardPlaceAuthDailyWorkPeriod ::= SEQUENCE {
//	    placeAuthPointerNewestRecord INTEGER(0..NoOfCardPlaceRecords -1),
//	    placeAuthStatusRecords SET SIZE (NoOfCardPlaceRecords) OF PlaceAuthStatusRecord
//	}
//
// Binary Layout:
//   - Bytes 0-1: placeAuthPointerNewestRecord
//   - Bytes 2..N: array of PlaceAuthStatusRecord (5 bytes each)
//
// The file is a fixed-size ring of record slots; unused slots are zero-filled
// and are kept, so that the file marshals back to its original length.
func (opts UnmarshalOptions) unmarshalPlacesAuthentication(data []byte) (*cardv1.PlacesAuthentication, error) {
	const (
		idxNewestRecordIndex = 0
		lenNewestRecordIndex = 2
		lenRecord            = 5
	)
	if len(data) < lenNewestRecordIndex {
		return nil, fmt.Errorf("invalid data length for CardPlaceAuthDailyWorkPeriod: got %d, want at least %d", len(data), lenNewestRecordIndex)
	}
	recordsData := data[lenNewestRecordIndex:]
	if len(recordsData)%lenRecord != 0 {
		return nil, fmt.Errorf("invalid records data length for CardPlaceAuthDailyWorkPeriod: got %d bytes, not a multiple of %d", len(recordsData), lenRecord)
	}
	var target cardv1.PlacesAuthentication
	target.SetNewestRecordIndex(int32(binary.BigEndian.Uint16(data[idxNewestRecordIndex:])))
	records := make([]*cardv1.PlacesAuthentication_Record, 0, len(recordsData)/lenRecord)
	for offset := 0; offset < len(recordsData); offset += lenRecord {
		record, err := opts.unmarshalPlaceAuthStatusRecord(recordsData[offset : offset+lenRecord])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal place auth status record %d: %w", offset/lenRecord, err)
		}
		records = append(records, record)
	}
	target.SetRecords(records)
	return &target, nil
}

// unmarshalPlaceAuthStatusRecord parses a single PlaceAuthStatusRecord.
//
// The data type `PlaceAuthStatusRecord` is specified in the Data Dictionary, Section 2.116b.
//
// ASN.1 Definition:
//
//	PlaceAuthStatusRecord ::= SEQUENCE {
//	    entryTime TimeReal,
//	    authenticationStatus PositionAuthenticationStatus
//	}
//
// Binary Layout (fixed length, 5 bytes):
//   - Bytes 0-3: entryTime
//   - Byte 4: authenticationStatus
func (opts UnmarshalOptions) unmarshalPlaceAuthStatusRecord(data []byte) (*cardv1.PlacesAuthentication_Record, error) {
	const (
		idxEntryTime            = 0
		idxAuthenticationStatus = 4
		lenRecord               = 5
		lenTimeReal             = 4
	)
	if len(data) != lenRecord {
		return nil, fmt.Errorf("invalid data length for PlaceAuthStatusRecord: got %d, want %d", len(data), lenRecord)
	}
	var record cardv1.PlacesAuthentication_Record
	entryTime, err := opts.UnmarshalTimeReal(data[idxEntryTime : idxEntryTime+lenTimeReal])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal entry time: %w", err)
	}
	record.SetEntryTime(entryTime)
	if status, err := dd.UnmarshalEnum[ddv1.PositionAuthenticationStatus](data[idxAuthenticationStatus]); err == nil {
		record.SetAuthenticationStatus(status)
	} else {
		record.SetAuthenticationStatus(ddv1.PositionAuthenticationStatus_POSITION_AUTHENTICATION_STATUS_UNRECOGNIZED)
		record.SetUnrecognizedAuthenticationStatus(int32(data[idxAuthenticationStatus]))
	}
	return &record, nil
}

// MarshalPlacesAuthentication marshals the EF_Places_Authentication file.
func (opts MarshalOptions) MarshalPlacesAuthentication(msg *cardv1.PlacesAuthentication) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	dst := binary.BigEndian.AppendUint16(nil, uint16(msg.GetNewestRecordIndex()))
	for i, record := range msg.GetRecords() {
		recordData, err := opts.marshalPlaceAuthStatusRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal place auth status record %d: %w", i, err)
		}
		dst = append(dst, recordData...)
	}
	return dst, nil
}

// marshalPlaceAuthStatusRecord marshals a single PlaceAuthStatusRecord.
func (opts MarshalOptions) marshalPlaceAuthStatusRecord(record *cardv1.PlacesAuthentication_Record) ([]byte, error) {
	const (
		idxEntryTime            = 0
		idxAuthenticationStatus = 4
		lenRecord               = 5
	)
	var canvas [lenRecord]byte
	if record == nil {
		return canvas[:], nil
	}
	timeBytes, err := opts.MarshalTimeReal(record.GetEntryTime())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry time: %w", err)
	}
	copy(canvas[idxEntryTime:], timeBytes)
	if record.GetAuthenticationStatus() == ddv1.PositionAuthenticationStatus_POSITION_AUTHENTICATION_STATUS_UNRECOGNIZED {
		canvas[idxAuthenticationStatus] = byte(record.GetUnrecognizedAuthenticationStatus())
	} else {
		canvas[idxAuthenticationStatus], _ = dd.MarshalEnum(record.GetAuthenticationStatus())
	}
	return canvas[:], nil
}

// anonymizePlacesAuthentication creates an anonymized copy of the
// EF_Places_Authentication file. The records hold an entry time and an
// authentication result, both of which are kept: the positions they qualify
// live in EF_Places and are anonymized there.
func (opts AnonymizeOptions) anonymizePlacesAuthentication(placesAuth *cardv1.PlacesAuthentication) *cardv1.PlacesAuthentication {
	if placesAuth == nil {
		return nil
	}
	result := &cardv1.PlacesAuthentication{}
	result.SetNewestRecordIndex(placesAuth.GetNewestRecordIndex())
	records := make([]*cardv1.PlacesAuthentication_Record, len(placesAuth.GetRecords()))
	for i, record := range placesAuth.GetRecords() {
		anonymized := &cardv1.PlacesAuthentication_Record{}
		anonymized.SetEntryTime(record.GetEntryTime())
		anonymized.SetAuthenticationStatus(record.GetAuthenticationStatus())
		if record.HasUnrecognizedAuthenticationStatus() {
			anonymized.SetUnrecognizedAuthenticationStatus(record.GetUnrecognizedAuthenticationStatus())
		}
		records[i] = anonymized
	}
	result.SetRecords(records)
	return result
}
