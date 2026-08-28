package card

import (
	"encoding/binary"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// unmarshalLoadUnloadOperations parses the EF_Load_Unload_Operations file.
//
// The data type `CardLoadUnloadOperations` is specified in the Data Dictionary, Section 2.24c.
//
// ASN.1 Definition:
//
//	CardLoadUnloadOperations ::= SEQUENCE {
//	    loadUnloadPointerNewestRecord INTEGER(0..NoOfLoadUnloadRecords -1),
//	    cardLoadUnloadRecords SET SIZE (NoOfLoadUnloadRecords) OF CardLoadUnloadRecord
//	}
//
// Binary Layout:
//   - Bytes 0-1: loadUnloadPointerNewestRecord
//   - Bytes 2..N: array of CardLoadUnloadRecord (20 bytes each)
//
// The file is a fixed-size ring of record slots; unused slots are zero-filled
// and are kept, so that the file marshals back to its original length.
func (opts UnmarshalOptions) unmarshalLoadUnloadOperations(data []byte) (*cardv1.LoadUnloadOperations, error) {
	const (
		idxNewestRecordIndex = 0
		lenNewestRecordIndex = 2
		lenRecord            = 20
	)
	if len(data) < lenNewestRecordIndex {
		return nil, fmt.Errorf("invalid data length for CardLoadUnloadOperations: got %d, want at least %d", len(data), lenNewestRecordIndex)
	}
	recordsData := data[lenNewestRecordIndex:]
	if len(recordsData)%lenRecord != 0 {
		return nil, fmt.Errorf("invalid records data length for CardLoadUnloadOperations: got %d bytes, not a multiple of %d", len(recordsData), lenRecord)
	}
	var target cardv1.LoadUnloadOperations
	target.SetNewestRecordIndex(int32(binary.BigEndian.Uint16(data[idxNewestRecordIndex:])))
	records := make([]*cardv1.LoadUnloadOperations_Record, 0, len(recordsData)/lenRecord)
	for offset := 0; offset < len(recordsData); offset += lenRecord {
		record, err := opts.unmarshalLoadUnloadRecord(recordsData[offset : offset+lenRecord])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal load/unload record %d: %w", offset/lenRecord, err)
		}
		records = append(records, record)
	}
	target.SetRecords(records)
	return &target, nil
}

// unmarshalLoadUnloadRecord parses a single CardLoadUnloadRecord.
//
// The data type `CardLoadUnloadRecord` is specified in the Data Dictionary, Section 2.24d.
//
// ASN.1 Definition:
//
//	CardLoadUnloadRecord ::= SEQUENCE {
//	    timeStamp TimeReal,
//	    operationType OperationType,
//	    gnssPlaceAuthRecord GNSSPlaceAuthRecord,
//	    vehicleOdometerValue OdometerShort
//	}
//
// Binary Layout (fixed length, 20 bytes):
//   - Bytes 0-3: timeStamp
//   - Byte 4: operationType
//   - Bytes 5-16: gnssPlaceAuthRecord
//   - Bytes 17-19: vehicleOdometerValue
func (opts UnmarshalOptions) unmarshalLoadUnloadRecord(data []byte) (*cardv1.LoadUnloadOperations_Record, error) {
	const (
		idxTimeStamp           = 0
		idxOperationType       = 4
		idxGnssPlaceAuthRecord = 5
		idxVehicleOdometer     = 17
		lenRecord              = 20
		lenTimeReal            = 4
		lenGNSSPlaceAuthRecord = 12
		lenOdometerShort       = 3
	)
	if len(data) != lenRecord {
		return nil, fmt.Errorf("invalid data length for CardLoadUnloadRecord: got %d, want %d", len(data), lenRecord)
	}
	var record cardv1.LoadUnloadOperations_Record
	timestamp, err := opts.UnmarshalTimeReal(data[idxTimeStamp : idxTimeStamp+lenTimeReal])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}
	record.SetTimestamp(timestamp)
	// An unused slot carries '00'H, which the regulation reserves.
	if operationType, err := dd.UnmarshalEnum[ddv1.OperationType](data[idxOperationType]); err == nil {
		record.SetOperationType(operationType)
	} else {
		record.SetOperationType(ddv1.OperationType_OPERATION_TYPE_UNRECOGNIZED)
		record.SetUnrecognizedOperationType(int32(data[idxOperationType]))
	}
	gnssPlaceAuthRecord, err := opts.UnmarshalGNSSPlaceAuthRecord(data[idxGnssPlaceAuthRecord : idxGnssPlaceAuthRecord+lenGNSSPlaceAuthRecord])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal GNSS place auth record: %w", err)
	}
	record.SetGnssPlaceAuthRecord(gnssPlaceAuthRecord)
	odometer, err := opts.UnmarshalOdometer(data[idxVehicleOdometer : idxVehicleOdometer+lenOdometerShort])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal vehicle odometer: %w", err)
	}
	record.SetVehicleOdometerKm(int32(odometer))
	return &record, nil
}

// MarshalLoadUnloadOperations marshals the EF_Load_Unload_Operations file.
func (opts MarshalOptions) MarshalLoadUnloadOperations(msg *cardv1.LoadUnloadOperations) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	dst := binary.BigEndian.AppendUint16(nil, uint16(msg.GetNewestRecordIndex()))
	for i, record := range msg.GetRecords() {
		recordData, err := opts.marshalLoadUnloadRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal load/unload record %d: %w", i, err)
		}
		dst = append(dst, recordData...)
	}
	return dst, nil
}

// marshalLoadUnloadRecord marshals a single CardLoadUnloadRecord.
func (opts MarshalOptions) marshalLoadUnloadRecord(record *cardv1.LoadUnloadOperations_Record) ([]byte, error) {
	const (
		idxTimeStamp           = 0
		idxOperationType       = 4
		idxGnssPlaceAuthRecord = 5
		idxVehicleOdometer     = 17
		lenRecord              = 20
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
	if record.GetOperationType() == ddv1.OperationType_OPERATION_TYPE_UNRECOGNIZED {
		canvas[idxOperationType] = byte(record.GetUnrecognizedOperationType())
	} else {
		canvas[idxOperationType], _ = dd.MarshalEnum(record.GetOperationType())
	}
	gnssBytes, err := opts.MarshalGNSSPlaceAuthRecord(record.GetGnssPlaceAuthRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GNSS place auth record: %w", err)
	}
	copy(canvas[idxGnssPlaceAuthRecord:], gnssBytes)
	odometerBytes, err := opts.MarshalOdometer(record.GetVehicleOdometerKm())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vehicle odometer: %w", err)
	}
	copy(canvas[idxVehicleOdometer:], odometerBytes)
	return canvas[:], nil
}

// anonymizeLoadUnloadOperations creates an anonymized copy of the
// EF_Load_Unload_Operations file: the operations and their sequence are
// preserved, the positions and odometer readings are replaced.
func (opts AnonymizeOptions) anonymizeLoadUnloadOperations(loadUnload *cardv1.LoadUnloadOperations) *cardv1.LoadUnloadOperations {
	if loadUnload == nil {
		return nil
	}
	ddOpts := dd.AnonymizeOptions{
		PreserveDistanceAndTrips: opts.PreserveDistanceAndTrips,
		PreserveTimestamps:       opts.PreserveTimestamps,
	}
	result := &cardv1.LoadUnloadOperations{}
	result.SetNewestRecordIndex(loadUnload.GetNewestRecordIndex())
	records := make([]*cardv1.LoadUnloadOperations_Record, len(loadUnload.GetRecords()))
	for i, record := range loadUnload.GetRecords() {
		anonymized := &cardv1.LoadUnloadOperations_Record{}
		anonymized.SetTimestamp(record.GetTimestamp())
		anonymized.SetOperationType(record.GetOperationType())
		if record.HasUnrecognizedOperationType() {
			anonymized.SetUnrecognizedOperationType(record.GetUnrecognizedOperationType())
		}
		anonymized.SetGnssPlaceAuthRecord(ddOpts.AnonymizeGNSSPlaceAuthRecord(record.GetGnssPlaceAuthRecord()))
		anonymized.SetVehicleOdometerKm(ddOpts.AnonymizeOdometerValue(record.GetVehicleOdometerKm()))
		records[i] = anonymized
	}
	result.SetRecords(records)
	return result
}
