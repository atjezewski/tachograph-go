package dd

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// UnmarshalVuOverspeedEventRecord parses a VuOverspeedEventRecord (Generation 1).
//
// The data type `VuOverSpeedingEventRecord` is specified in the Data Dictionary, Section 2.215.
//
// ASN.1 Specification (Gen1):
//
//	VuOverSpeedingEventRecord ::= SEQUENCE {
//	    eventType                       EventFaultType,               -- 1 byte
//	    eventRecordPurpose              EventFaultRecordPurpose,      -- 1 byte
//	    eventBeginTime                  TimeReal,                     -- 4 bytes
//	    eventEndTime                    TimeReal,                     -- 4 bytes
//	    maxSpeedValue                   SpeedMax,                     -- 1 byte
//	    averageSpeedValue               SpeedAverage,                 -- 1 byte
//	    cardNumberDriverSlotBegin       FullCardNumber,               -- 18 bytes
//	    similarEventsNumber             SimilarEventsNumber           -- 1 byte
//	}
func (opts UnmarshalOptions) UnmarshalVuOverspeedEventRecord(data []byte) (*ddv1.VuOverspeedEventRecord, error) {
	const (
		idxEventType                 = 0
		idxEventRecordPurpose        = 1
		idxEventBeginTime            = 2
		lenEventBeginTime            = 4
		idxEventEndTime              = 6
		lenEventEndTime              = 4
		idxMaxSpeedValue             = 10
		idxAverageSpeedValue         = 11
		idxCardNumberDriverSlotBegin = 12
		lenCardNumber                = 18
		idxSimilarEventsNumber       = 30
		lenVuOverspeedEventRecord    = 31
	)

	if len(data) != lenVuOverspeedEventRecord {
		return nil, fmt.Errorf("invalid length for VuOverspeedEventRecord: got %d, want %d", len(data), lenVuOverspeedEventRecord)
	}

	record := &ddv1.VuOverspeedEventRecord{}
	if opts.PreserveRawData {
		record.SetRawData(data)
	}

	// Parse eventType (1 byte)
	eventType, unrecognizedEventType := opts.parseEventFaultType(data[idxEventType])
	if eventType != ddv1.EventFaultType_EVENT_FAULT_TYPE_UNSPECIFIED {
		record.SetEventType(eventType)
	}
	if unrecognizedEventType != 0 {
		record.SetUnrecognizedEventType(unrecognizedEventType)
	}

	// Parse eventRecordPurpose (1 byte)
	recordPurpose, unrecognizedPurpose := opts.parseEventFaultRecordPurpose(data[idxEventRecordPurpose])
	if recordPurpose != ddv1.EventFaultRecordPurpose_EVENT_FAULT_RECORD_PURPOSE_UNSPECIFIED {
		record.SetRecordPurpose(recordPurpose)
	}
	if unrecognizedPurpose != 0 {
		record.SetUnrecognizedRecordPurpose(unrecognizedPurpose)
	}

	// Parse eventBeginTime (4 bytes)
	beginTime, err := opts.UnmarshalTimeReal(data[idxEventBeginTime : idxEventBeginTime+lenEventBeginTime])
	if err != nil {
		return nil, fmt.Errorf("failed to parse event begin time: %w", err)
	}
	record.SetBeginTime(beginTime)

	// Parse eventEndTime (4 bytes)
	endTime, err := opts.UnmarshalTimeReal(data[idxEventEndTime : idxEventEndTime+lenEventEndTime])
	if err != nil {
		return nil, fmt.Errorf("failed to parse event end time: %w", err)
	}
	record.SetEndTime(endTime)

	// Parse maxSpeedValue (1 byte)
	record.SetMaxSpeedKmh(int32(data[idxMaxSpeedValue]))

	// Parse averageSpeedValue (1 byte)
	record.SetAverageSpeedKmh(int32(data[idxAverageSpeedValue]))

	// Parse cardNumberDriverSlotBegin (18 bytes)
	cardDriverBegin, err := opts.UnmarshalFullCardNumber(data[idxCardNumberDriverSlotBegin : idxCardNumberDriverSlotBegin+lenCardNumber])
	if err != nil {
		return nil, fmt.Errorf("failed to parse driver card begin: %w", err)
	}
	record.SetCardNumberDriverSlotBegin(cardDriverBegin)

	// Parse similarEventsNumber (1 byte)
	record.SetSimilarEventsNumber(int32(data[idxSimilarEventsNumber]))

	return record, nil
}

// MarshalVuOverspeedEventRecord marshals a VuOverspeedEventRecord to binary format (Generation 1).
func (opts MarshalOptions) MarshalVuOverspeedEventRecord(record *ddv1.VuOverspeedEventRecord) ([]byte, error) {
	const lenVuOverspeedEventRecord = 31

	if record == nil {
		return nil, fmt.Errorf("record cannot be nil")
	}

	// Use raw data painting strategy if available
	var canvas [lenVuOverspeedEventRecord]byte
	if record.HasRawData() {
		if len(record.GetRawData()) != lenVuOverspeedEventRecord {
			return nil, fmt.Errorf(
				"invalid raw_data length for VuOverspeedEventRecord: got %d, want %d",
				len(record.GetRawData()), lenVuOverspeedEventRecord,
			)
		}
		copy(canvas[:], record.GetRawData())
	}

	// Paint semantic values over the canvas
	const (
		idxEventType                 = 0
		idxEventRecordPurpose        = 1
		idxEventBeginTime            = 2
		lenEventBeginTime            = 4
		idxEventEndTime              = 6
		lenEventEndTime              = 4
		idxMaxSpeedValue             = 10
		idxAverageSpeedValue         = 11
		idxCardNumberDriverSlotBegin = 12
		lenCardNumber                = 18
		idxSimilarEventsNumber       = 30
	)

	// Marshal eventType (1 byte)
	eventType, err := opts.marshalEventFaultType(record.GetEventType(), record.GetUnrecognizedEventType())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event type: %w", err)
	}
	canvas[idxEventType] = eventType

	// Marshal eventRecordPurpose (1 byte)
	recordPurpose, err := opts.marshalEventFaultRecordPurpose(record.GetRecordPurpose(), record.GetUnrecognizedRecordPurpose())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event record purpose: %w", err)
	}
	canvas[idxEventRecordPurpose] = recordPurpose

	// Marshal eventBeginTime (4 bytes)
	beginTime, err := opts.MarshalTimeReal(record.GetBeginTime())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal begin time: %w", err)
	}
	copy(canvas[idxEventBeginTime:idxEventBeginTime+lenEventBeginTime], beginTime)

	// Marshal eventEndTime (4 bytes)
	endTime, err := opts.MarshalTimeReal(record.GetEndTime())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal end time: %w", err)
	}
	copy(canvas[idxEventEndTime:idxEventEndTime+lenEventEndTime], endTime)

	// Marshal maxSpeedValue (1 byte)
	canvas[idxMaxSpeedValue] = byte(record.GetMaxSpeedKmh())

	// Marshal averageSpeedValue (1 byte)
	canvas[idxAverageSpeedValue] = byte(record.GetAverageSpeedKmh())

	// Marshal cardNumberDriverSlotBegin (18 bytes)
	cardDriverBegin, err := opts.MarshalFullCardNumber(record.GetCardNumberDriverSlotBegin())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal driver card begin: %w", err)
	}
	copy(canvas[idxCardNumberDriverSlotBegin:idxCardNumberDriverSlotBegin+lenCardNumber], cardDriverBegin)

	// Marshal similarEventsNumber (1 byte)
	canvas[idxSimilarEventsNumber] = byte(record.GetSimilarEventsNumber())

	return canvas[:], nil
}

// AnonymizeVuOverspeedEventRecord anonymizes a VU overspeed event record.
func (opts AnonymizeOptions) AnonymizeVuOverspeedEventRecord(rec *ddv1.VuOverspeedEventRecord) *ddv1.VuOverspeedEventRecord {
	if rec == nil {
		return nil
	}

	result := proto.Clone(rec).(*ddv1.VuOverspeedEventRecord)

	// Anonymize timestamps
	result.SetBeginTime(opts.AnonymizeTimestamp(rec.GetBeginTime()))
	result.SetEndTime(opts.AnonymizeTimestamp(rec.GetEndTime()))

	// Anonymize card number
	result.SetCardNumberDriverSlotBegin(opts.AnonymizeFullCardNumber(rec.GetCardNumberDriverSlotBegin()))

	// Speed values are not PII - keep as-is
	// (max_speed_kmh, average_speed_kmh, similar_events_number are not personally identifiable)

	// Clear raw_data
	result.ClearRawData()

	return result
}
