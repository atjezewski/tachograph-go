package card

import (
	"encoding/binary"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// unmarshalGnssPlacesAuthentication parses the EF_GNSS_Places_Authentication
// file, which records whether each position in EF_GNSS_Places was authenticated
// by the GNSS receiver.
//
// The data type `GNSSAuthAccumulatedDriving` is specified in the Data Dictionary, Section 2.79a.
//
// ASN.1 Definition:
//
//	GNSSAuthAccumulatedDriving ::= SEQUENCE {
//	    gnssAuthADPointerNewestRecord INTEGER(0..NoOfGNSSADRecords -1),
//	    gnssAuthStatusADRecords SET SIZE (NoOfGNSSADRecords) OF GNSSAuthStatusADRecord
//	}
//
// Binary Layout:
//   - Bytes 0-1: gnssAuthADPointerNewestRecord
//   - Bytes 2..N: array of GNSSAuthStatusADRecord (5 bytes each)
//
// The file is a fixed-size ring of record slots; unused slots are zero-filled
// and are kept, so that the file marshals back to its original length.
func (opts UnmarshalOptions) unmarshalGnssPlacesAuthentication(data []byte) (*cardv1.GnssPlacesAuthentication, error) {
	const (
		idxNewestRecordIndex = 0
		lenNewestRecordIndex = 2
		lenRecord            = 5
	)
	if len(data) < lenNewestRecordIndex {
		return nil, fmt.Errorf("invalid data length for GNSSAuthAccumulatedDriving: got %d, want at least %d", len(data), lenNewestRecordIndex)
	}
	recordsData := data[lenNewestRecordIndex:]
	if len(recordsData)%lenRecord != 0 {
		return nil, fmt.Errorf("invalid records data length for GNSSAuthAccumulatedDriving: got %d bytes, not a multiple of %d", len(recordsData), lenRecord)
	}
	var target cardv1.GnssPlacesAuthentication
	target.SetNewestRecordIndex(int32(binary.BigEndian.Uint16(data[idxNewestRecordIndex:])))
	records := make([]*cardv1.GnssPlacesAuthentication_Record, 0, len(recordsData)/lenRecord)
	for offset := 0; offset < len(recordsData); offset += lenRecord {
		record, err := opts.unmarshalGnssAuthStatusADRecord(recordsData[offset : offset+lenRecord])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal GNSS auth status record %d: %w", offset/lenRecord, err)
		}
		records = append(records, record)
	}
	target.SetRecords(records)
	return &target, nil
}

// unmarshalGnssAuthStatusADRecord parses a single GNSSAuthStatusADRecord.
//
// The data type `GNSSAuthStatusADRecord` is specified in the Data Dictionary, Section 2.79b.
//
// ASN.1 Definition:
//
//	GNSSAuthStatusADRecord ::= SEQUENCE {
//	    timeStamp TimeReal,
//	    authenticationStatus PositionAuthenticationStatus
//	}
//
// Binary Layout (fixed length, 5 bytes):
//   - Bytes 0-3: timeStamp
//   - Byte 4: authenticationStatus
func (opts UnmarshalOptions) unmarshalGnssAuthStatusADRecord(data []byte) (*cardv1.GnssPlacesAuthentication_Record, error) {
	const (
		idxTimeStamp            = 0
		idxAuthenticationStatus = 4
		lenRecord               = 5
		lenTimeReal             = 4
	)
	if len(data) != lenRecord {
		return nil, fmt.Errorf("invalid data length for GNSSAuthStatusADRecord: got %d, want %d", len(data), lenRecord)
	}
	var record cardv1.GnssPlacesAuthentication_Record
	timestamp, err := opts.UnmarshalTimeReal(data[idxTimeStamp : idxTimeStamp+lenTimeReal])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}
	record.SetTimestamp(timestamp)
	if status, err := dd.UnmarshalEnum[ddv1.PositionAuthenticationStatus](data[idxAuthenticationStatus]); err == nil {
		record.SetAuthenticationStatus(status)
	} else {
		record.SetAuthenticationStatus(ddv1.PositionAuthenticationStatus_POSITION_AUTHENTICATION_STATUS_UNRECOGNIZED)
		record.SetUnrecognizedAuthenticationStatus(int32(data[idxAuthenticationStatus]))
	}
	return &record, nil
}

// MarshalGnssPlacesAuthentication marshals the EF_GNSS_Places_Authentication file.
func (opts MarshalOptions) MarshalGnssPlacesAuthentication(msg *cardv1.GnssPlacesAuthentication) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	dst := binary.BigEndian.AppendUint16(nil, uint16(msg.GetNewestRecordIndex()))
	for i, record := range msg.GetRecords() {
		recordData, err := opts.marshalGnssAuthStatusADRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal GNSS auth status record %d: %w", i, err)
		}
		dst = append(dst, recordData...)
	}
	return dst, nil
}

// marshalGnssAuthStatusADRecord marshals a single GNSSAuthStatusADRecord.
func (opts MarshalOptions) marshalGnssAuthStatusADRecord(record *cardv1.GnssPlacesAuthentication_Record) ([]byte, error) {
	const (
		idxTimeStamp            = 0
		idxAuthenticationStatus = 4
		lenRecord               = 5
	)
	var canvas [lenRecord]byte
	if record == nil {
		return canvas[:], nil
	}
	timeBytes, err := opts.MarshalTimeReal(record.GetTimestamp())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal timestamp: %w", err)
	}
	copy(canvas[idxTimeStamp:], timeBytes)
	if record.GetAuthenticationStatus() == ddv1.PositionAuthenticationStatus_POSITION_AUTHENTICATION_STATUS_UNRECOGNIZED {
		canvas[idxAuthenticationStatus] = byte(record.GetUnrecognizedAuthenticationStatus())
	} else {
		canvas[idxAuthenticationStatus], _ = dd.MarshalEnum(record.GetAuthenticationStatus())
	}
	return canvas[:], nil
}

// anonymizeGnssPlacesAuthentication creates an anonymized copy of the
// EF_GNSS_Places_Authentication file. The records hold a timestamp and an
// authentication result, both of which are kept: the positions they qualify
// live in EF_GNSS_Places and are anonymized there.
func (opts AnonymizeOptions) anonymizeGnssPlacesAuthentication(gnssAuth *cardv1.GnssPlacesAuthentication) *cardv1.GnssPlacesAuthentication {
	if gnssAuth == nil {
		return nil
	}
	result := &cardv1.GnssPlacesAuthentication{}
	result.SetNewestRecordIndex(gnssAuth.GetNewestRecordIndex())
	records := make([]*cardv1.GnssPlacesAuthentication_Record, len(gnssAuth.GetRecords()))
	for i, record := range gnssAuth.GetRecords() {
		anonymized := &cardv1.GnssPlacesAuthentication_Record{}
		anonymized.SetTimestamp(record.GetTimestamp())
		anonymized.SetAuthenticationStatus(record.GetAuthenticationStatus())
		if record.HasUnrecognizedAuthenticationStatus() {
			anonymized.SetUnrecognizedAuthenticationStatus(record.GetUnrecognizedAuthenticationStatus())
		}
		records[i] = anonymized
	}
	result.SetRecords(records)
	return result
}
