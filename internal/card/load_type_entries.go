package card

import (
	"encoding/binary"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// unmarshalLoadTypeEntries parses the EF_Load_Type_Entries file.
//
// The data type `CardLoadTypeEntries` is specified in the Data Dictionary, Section 2.24a.
//
// ASN.1 Definition:
//
//	CardLoadTypeEntries ::= SEQUENCE {
//	    loadTypeEntryPointerNewestRecord INTEGER(0..NoOfLoadTypeEntryRecords-1),
//	    cardLoadTypeEntryRecords SET SIZE (NoOfLoadTypeEntryRecords) OF CardLoadTypeEntryRecord
//	}
//
// Binary Layout:
//   - Bytes 0-1: loadTypeEntryPointerNewestRecord
//   - Bytes 2..N: array of CardLoadTypeEntryRecord (5 bytes each)
//
// The file is a fixed-size ring of record slots; unused slots are zero-filled
// and are kept, so that the file marshals back to its original length.
func (opts UnmarshalOptions) unmarshalLoadTypeEntries(data []byte) (*cardv1.LoadTypeEntries, error) {
	const (
		idxNewestRecordIndex = 0
		lenNewestRecordIndex = 2
		lenRecord            = 5
	)
	if len(data) < lenNewestRecordIndex {
		return nil, fmt.Errorf("invalid data length for CardLoadTypeEntries: got %d, want at least %d", len(data), lenNewestRecordIndex)
	}
	recordsData := data[lenNewestRecordIndex:]
	if len(recordsData)%lenRecord != 0 {
		return nil, fmt.Errorf("invalid records data length for CardLoadTypeEntries: got %d bytes, not a multiple of %d", len(recordsData), lenRecord)
	}
	var target cardv1.LoadTypeEntries
	target.SetNewestRecordIndex(int32(binary.BigEndian.Uint16(data[idxNewestRecordIndex:])))
	records := make([]*cardv1.LoadTypeEntries_Record, 0, len(recordsData)/lenRecord)
	for offset := 0; offset < len(recordsData); offset += lenRecord {
		record, err := opts.unmarshalLoadTypeEntryRecord(recordsData[offset : offset+lenRecord])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal load type entry record %d: %w", offset/lenRecord, err)
		}
		records = append(records, record)
	}
	target.SetRecords(records)
	return &target, nil
}

// unmarshalLoadTypeEntryRecord parses a single CardLoadTypeEntryRecord.
//
// The data type `CardLoadTypeEntryRecord` is specified in the Data Dictionary, Section 2.24b.
//
// ASN.1 Definition:
//
//	CardLoadTypeEntryRecord ::= SEQUENCE {
//	    timeStamp TimeReal,
//	    loadTypeEntered LoadType
//	}
//
// Binary Layout (fixed length, 5 bytes):
//   - Bytes 0-3: timeStamp
//   - Byte 4: loadTypeEntered
func (opts UnmarshalOptions) unmarshalLoadTypeEntryRecord(data []byte) (*cardv1.LoadTypeEntries_Record, error) {
	const (
		idxTimeStamp       = 0
		idxLoadTypeEntered = 4
		lenRecord          = 5
		lenTimeReal        = 4
	)
	if len(data) != lenRecord {
		return nil, fmt.Errorf("invalid data length for CardLoadTypeEntryRecord: got %d, want %d", len(data), lenRecord)
	}
	var record cardv1.LoadTypeEntries_Record
	timestamp, err := opts.UnmarshalTimeReal(data[idxTimeStamp : idxTimeStamp+lenTimeReal])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}
	record.SetTimestamp(timestamp)
	if loadType, err := dd.UnmarshalEnum[ddv1.LoadType](data[idxLoadTypeEntered]); err == nil {
		record.SetLoadTypeEntered(loadType)
	} else {
		record.SetLoadTypeEntered(ddv1.LoadType_LOAD_TYPE_UNRECOGNIZED)
		record.SetUnrecognizedLoadTypeEntered(int32(data[idxLoadTypeEntered]))
	}
	return &record, nil
}

// MarshalLoadTypeEntries marshals the EF_Load_Type_Entries file.
func (opts MarshalOptions) MarshalLoadTypeEntries(msg *cardv1.LoadTypeEntries) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	dst := binary.BigEndian.AppendUint16(nil, uint16(msg.GetNewestRecordIndex()))
	for i, record := range msg.GetRecords() {
		recordData, err := opts.marshalLoadTypeEntryRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal load type entry record %d: %w", i, err)
		}
		dst = append(dst, recordData...)
	}
	return dst, nil
}

// marshalLoadTypeEntryRecord marshals a single CardLoadTypeEntryRecord.
func (opts MarshalOptions) marshalLoadTypeEntryRecord(record *cardv1.LoadTypeEntries_Record) ([]byte, error) {
	const (
		idxTimeStamp       = 0
		idxLoadTypeEntered = 4
		lenRecord          = 5
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
	if record.GetLoadTypeEntered() == ddv1.LoadType_LOAD_TYPE_UNRECOGNIZED {
		canvas[idxLoadTypeEntered] = byte(record.GetUnrecognizedLoadTypeEntered())
	} else {
		canvas[idxLoadTypeEntered], _ = dd.MarshalEnum(record.GetLoadTypeEntered())
	}
	return canvas[:], nil
}

// anonymizeLoadTypeEntries creates an anonymized copy of the
// EF_Load_Type_Entries file. The records carry no personal data beyond their
// timing, so they are copied as they are.
func (opts AnonymizeOptions) anonymizeLoadTypeEntries(loadTypeEntries *cardv1.LoadTypeEntries) *cardv1.LoadTypeEntries {
	if loadTypeEntries == nil {
		return nil
	}
	result := &cardv1.LoadTypeEntries{}
	result.SetNewestRecordIndex(loadTypeEntries.GetNewestRecordIndex())
	records := make([]*cardv1.LoadTypeEntries_Record, len(loadTypeEntries.GetRecords()))
	for i, record := range loadTypeEntries.GetRecords() {
		anonymized := &cardv1.LoadTypeEntries_Record{}
		anonymized.SetTimestamp(record.GetTimestamp())
		anonymized.SetLoadTypeEntered(record.GetLoadTypeEntered())
		if record.HasUnrecognizedLoadTypeEntered() {
			anonymized.SetUnrecognizedLoadTypeEntered(record.GetUnrecognizedLoadTypeEntered())
		}
		records[i] = anonymized
	}
	result.SetRecords(records)
	return result
}
