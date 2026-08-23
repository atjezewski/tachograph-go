package card

import (
	"encoding/binary"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// unmarshalBorderCrossings parses the EF_Border_Crossings file.
//
// The data type `CardBorderCrossings` is specified in the Data Dictionary, Section 2.11a.
//
// ASN.1 Definition:
//
//	CardBorderCrossings ::= SEQUENCE {
//	    borderCrossingPointerNewestRecord INTEGER (0..NoOfBorderCrossingRecords -1),
//	    cardBorderCrossingRecords SET SIZE (NoOfBorderCrossingRecords) OF CardBorderCrossingRecord
//	}
//
// Binary Layout:
//   - Bytes 0-1: borderCrossingPointerNewestRecord
//   - Bytes 2..N: array of CardBorderCrossingRecord (17 bytes each)
//
// The file is a fixed-size ring of record slots; unused slots are zero-filled
// and are kept, so that the file marshals back to its original length.
func (opts UnmarshalOptions) unmarshalBorderCrossings(data []byte) (*cardv1.BorderCrossings, error) {
	const (
		idxNewestRecordIndex = 0
		lenNewestRecordIndex = 2
		lenRecord            = 17
	)
	if len(data) < lenNewestRecordIndex {
		return nil, fmt.Errorf("invalid data length for CardBorderCrossings: got %d, want at least %d", len(data), lenNewestRecordIndex)
	}
	recordsData := data[lenNewestRecordIndex:]
	if len(recordsData)%lenRecord != 0 {
		return nil, fmt.Errorf("invalid records data length for CardBorderCrossings: got %d bytes, not a multiple of %d", len(recordsData), lenRecord)
	}
	var target cardv1.BorderCrossings
	target.SetNewestRecordIndex(int32(binary.BigEndian.Uint16(data[idxNewestRecordIndex:])))
	records := make([]*cardv1.BorderCrossings_Record, 0, len(recordsData)/lenRecord)
	for offset := 0; offset < len(recordsData); offset += lenRecord {
		record, err := opts.unmarshalBorderCrossingRecord(recordsData[offset : offset+lenRecord])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal border crossing record %d: %w", offset/lenRecord, err)
		}
		records = append(records, record)
	}
	target.SetRecords(records)
	return &target, nil
}

// unmarshalBorderCrossingRecord parses a single CardBorderCrossingRecord.
//
// The data type `CardBorderCrossingRecord` is specified in the Data Dictionary, Section 2.11b.
//
// ASN.1 Definition:
//
//	CardBorderCrossingRecord ::= SEQUENCE {
//	    countryLeft NationNumeric,
//	    countryEntered NationNumeric,
//	    gnssPlaceAuthRecord GNSSPlaceAuthRecord,
//	    vehicleOdometerValue OdometerShort
//	}
//
// Binary Layout (fixed length, 17 bytes):
//   - Byte 0: countryLeft
//   - Byte 1: countryEntered
//   - Bytes 2-13: gnssPlaceAuthRecord
//   - Bytes 14-16: vehicleOdometerValue
func (opts UnmarshalOptions) unmarshalBorderCrossingRecord(data []byte) (*cardv1.BorderCrossings_Record, error) {
	const (
		idxCountryLeft         = 0
		idxCountryEntered      = 1
		idxGnssPlaceAuthRecord = 2
		idxVehicleOdometer     = 14
		lenRecord              = 17
		lenGNSSPlaceAuthRecord = 12
		lenOdometerShort       = 3
	)
	if len(data) != lenRecord {
		return nil, fmt.Errorf("invalid data length for CardBorderCrossingRecord: got %d, want %d", len(data), lenRecord)
	}
	var record cardv1.BorderCrossings_Record
	if countryLeft, err := dd.UnmarshalEnum[ddv1.NationNumeric](data[idxCountryLeft]); err == nil {
		record.SetCountryLeft(countryLeft)
	} else {
		record.SetCountryLeft(ddv1.NationNumeric_NATION_NUMERIC_UNRECOGNIZED)
		record.SetUnrecognizedCountryLeft(int32(data[idxCountryLeft]))
	}
	if countryEntered, err := dd.UnmarshalEnum[ddv1.NationNumeric](data[idxCountryEntered]); err == nil {
		record.SetCountryEntered(countryEntered)
	} else {
		record.SetCountryEntered(ddv1.NationNumeric_NATION_NUMERIC_UNRECOGNIZED)
		record.SetUnrecognizedCountryEntered(int32(data[idxCountryEntered]))
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

// MarshalBorderCrossings marshals the EF_Border_Crossings file.
func (opts MarshalOptions) MarshalBorderCrossings(msg *cardv1.BorderCrossings) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	dst := binary.BigEndian.AppendUint16(nil, uint16(msg.GetNewestRecordIndex()))
	for i, record := range msg.GetRecords() {
		recordData, err := opts.marshalBorderCrossingRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal border crossing record %d: %w", i, err)
		}
		dst = append(dst, recordData...)
	}
	return dst, nil
}

// marshalBorderCrossingRecord marshals a single CardBorderCrossingRecord.
func (opts MarshalOptions) marshalBorderCrossingRecord(record *cardv1.BorderCrossings_Record) ([]byte, error) {
	const (
		idxGnssPlaceAuthRecord = 2
		idxVehicleOdometer     = 14
		lenRecord              = 17
	)
	var canvas [lenRecord]byte
	if record == nil {
		return canvas[:], nil
	}
	if record.GetCountryLeft() == ddv1.NationNumeric_NATION_NUMERIC_UNRECOGNIZED {
		canvas[0] = byte(record.GetUnrecognizedCountryLeft())
	} else {
		canvas[0], _ = dd.MarshalEnum(record.GetCountryLeft())
	}
	if record.GetCountryEntered() == ddv1.NationNumeric_NATION_NUMERIC_UNRECOGNIZED {
		canvas[1] = byte(record.GetUnrecognizedCountryEntered())
	} else {
		canvas[1], _ = dd.MarshalEnum(record.GetCountryEntered())
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

// anonymizeBorderCrossings creates an anonymized copy of the EF_Border_Crossings
// file: the countries crossed and the sequence of crossings are preserved, the
// positions and odometer readings that identify the vehicle are replaced.
func (opts AnonymizeOptions) anonymizeBorderCrossings(borderCrossings *cardv1.BorderCrossings) *cardv1.BorderCrossings {
	if borderCrossings == nil {
		return nil
	}
	ddOpts := dd.AnonymizeOptions{
		PreserveDistanceAndTrips: opts.PreserveDistanceAndTrips,
		PreserveTimestamps:       opts.PreserveTimestamps,
	}
	result := &cardv1.BorderCrossings{}
	result.SetNewestRecordIndex(borderCrossings.GetNewestRecordIndex())
	records := make([]*cardv1.BorderCrossings_Record, len(borderCrossings.GetRecords()))
	for i, record := range borderCrossings.GetRecords() {
		anonymized := &cardv1.BorderCrossings_Record{}
		anonymized.SetCountryLeft(record.GetCountryLeft())
		if record.HasUnrecognizedCountryLeft() {
			anonymized.SetUnrecognizedCountryLeft(record.GetUnrecognizedCountryLeft())
		}
		anonymized.SetCountryEntered(record.GetCountryEntered())
		if record.HasUnrecognizedCountryEntered() {
			anonymized.SetUnrecognizedCountryEntered(record.GetUnrecognizedCountryEntered())
		}
		anonymized.SetGnssPlaceAuthRecord(ddOpts.AnonymizeGNSSPlaceAuthRecord(record.GetGnssPlaceAuthRecord()))
		anonymized.SetVehicleOdometerKm(ddOpts.AnonymizeOdometerValue(record.GetVehicleOdometerKm()))
		records[i] = anonymized
	}
	result.SetRecords(records)
	return result
}
