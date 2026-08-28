package dd

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// UnmarshalVuEventRecord parses a VuEventRecord (Generation 1).
//
// The data type `VuEventRecord` is specified in the Data Dictionary, Section 2.198.
//
// ASN.1 Specification (Gen1):
//
//	VuEventRecord ::= SEQUENCE {
//	    eventType                       EventFaultType,               -- 1 byte
//	    eventRecordPurpose              EventFaultRecordPurpose,      -- 1 byte
//	    eventBeginTime                  TimeReal,                     -- 4 bytes
//	    eventEndTime                    TimeReal,                     -- 4 bytes
//	    cardNumberDriverSlotBegin       FullCardNumber,               -- 18 bytes
//	    cardNumberCodriverSlotBegin     FullCardNumber,               -- 18 bytes
//	    cardNumberDriverSlotEnd         FullCardNumber,               -- 18 bytes
//	    cardNumberCodriverSlotEnd       FullCardNumber,               -- 18 bytes
//	    similarEventsNumber             SimilarEventsNumber           -- 1 byte
//	}
func (opts UnmarshalOptions) UnmarshalVuEventRecord(data []byte) (*ddv1.VuEventRecord, error) {
	const (
		idxEventType                   = 0
		idxEventRecordPurpose          = 1
		idxEventBeginTime              = 2
		lenEventBeginTime              = 4
		idxEventEndTime                = 6
		lenEventEndTime                = 4
		idxCardNumberDriverSlotBegin   = 10
		lenCardNumber                  = 18
		idxCardNumberCodriverSlotBegin = 28
		idxCardNumberDriverSlotEnd     = 46
		idxCardNumberCodriverSlotEnd   = 64
		idxSimilarEventsNumber         = 82
		lenVuEventRecord               = 83
	)

	if len(data) != lenVuEventRecord {
		return nil, fmt.Errorf("invalid length for VuEventRecord: got %d, want %d", len(data), lenVuEventRecord)
	}

	record := &ddv1.VuEventRecord{}
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

	// Parse cardNumberDriverSlotBegin (18 bytes)
	cardDriverBegin, err := opts.UnmarshalFullCardNumber(data[idxCardNumberDriverSlotBegin : idxCardNumberDriverSlotBegin+lenCardNumber])
	if err != nil {
		return nil, fmt.Errorf("failed to parse driver card begin: %w", err)
	}
	record.SetCardNumberDriverSlotBegin(cardDriverBegin)

	// Parse cardNumberCodriverSlotBegin (18 bytes)
	cardCodriverBegin, err := opts.UnmarshalFullCardNumber(data[idxCardNumberCodriverSlotBegin : idxCardNumberCodriverSlotBegin+lenCardNumber])
	if err != nil {
		return nil, fmt.Errorf("failed to parse codriver card begin: %w", err)
	}
	record.SetCardNumberCodriverSlotBegin(cardCodriverBegin)

	// Parse cardNumberDriverSlotEnd (18 bytes)
	cardDriverEnd, err := opts.UnmarshalFullCardNumber(data[idxCardNumberDriverSlotEnd : idxCardNumberDriverSlotEnd+lenCardNumber])
	if err != nil {
		return nil, fmt.Errorf("failed to parse driver card end: %w", err)
	}
	record.SetCardNumberDriverSlotEnd(cardDriverEnd)

	// Parse cardNumberCodriverSlotEnd (18 bytes)
	cardCodriverEnd, err := opts.UnmarshalFullCardNumber(data[idxCardNumberCodriverSlotEnd : idxCardNumberCodriverSlotEnd+lenCardNumber])
	if err != nil {
		return nil, fmt.Errorf("failed to parse codriver card end: %w", err)
	}
	record.SetCardNumberCodriverSlotEnd(cardCodriverEnd)

	// Parse similarEventsNumber (1 byte)
	record.SetSimilarEventsNumber(int32(data[idxSimilarEventsNumber]))

	return record, nil
}

// MarshalVuEventRecord marshals a VuEventRecord to binary format (Generation 1).
func (opts MarshalOptions) MarshalVuEventRecord(record *ddv1.VuEventRecord) ([]byte, error) {
	const lenVuEventRecord = 83

	if record == nil {
		return nil, fmt.Errorf("record cannot be nil")
	}

	// Use raw data painting strategy if available
	var canvas [lenVuEventRecord]byte
	if record.HasRawData() {
		if len(record.GetRawData()) != lenVuEventRecord {
			return nil, fmt.Errorf(
				"invalid raw_data length for VuEventRecord: got %d, want %d",
				len(record.GetRawData()), lenVuEventRecord,
			)
		}
		copy(canvas[:], record.GetRawData())
	}

	// Paint semantic values over the canvas
	const (
		idxEventType                   = 0
		idxEventRecordPurpose          = 1
		idxEventBeginTime              = 2
		lenEventBeginTime              = 4
		idxEventEndTime                = 6
		lenEventEndTime                = 4
		idxCardNumberDriverSlotBegin   = 10
		lenCardNumber                  = 18
		idxCardNumberCodriverSlotBegin = 28
		idxCardNumberDriverSlotEnd     = 46
		idxCardNumberCodriverSlotEnd   = 64
		idxSimilarEventsNumber         = 82
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

	// Marshal cardNumberDriverSlotBegin (18 bytes)
	cardDriverBegin, err := opts.MarshalFullCardNumber(record.GetCardNumberDriverSlotBegin())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal driver card begin: %w", err)
	}
	copy(canvas[idxCardNumberDriverSlotBegin:idxCardNumberDriverSlotBegin+lenCardNumber], cardDriverBegin)

	// Marshal cardNumberCodriverSlotBegin (18 bytes)
	cardCodriverBegin, err := opts.MarshalFullCardNumber(record.GetCardNumberCodriverSlotBegin())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal codriver card begin: %w", err)
	}
	copy(canvas[idxCardNumberCodriverSlotBegin:idxCardNumberCodriverSlotBegin+lenCardNumber], cardCodriverBegin)

	// Marshal cardNumberDriverSlotEnd (18 bytes)
	cardDriverEnd, err := opts.MarshalFullCardNumber(record.GetCardNumberDriverSlotEnd())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal driver card end: %w", err)
	}
	copy(canvas[idxCardNumberDriverSlotEnd:idxCardNumberDriverSlotEnd+lenCardNumber], cardDriverEnd)

	// Marshal cardNumberCodriverSlotEnd (18 bytes)
	cardCodriverEnd, err := opts.MarshalFullCardNumber(record.GetCardNumberCodriverSlotEnd())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal codriver card end: %w", err)
	}
	copy(canvas[idxCardNumberCodriverSlotEnd:idxCardNumberCodriverSlotEnd+lenCardNumber], cardCodriverEnd)

	// Marshal similarEventsNumber (1 byte)
	canvas[idxSimilarEventsNumber] = byte(record.GetSimilarEventsNumber())

	return canvas[:], nil
}

// AnonymizeVuEventRecord anonymizes a VU event record.
func (opts AnonymizeOptions) AnonymizeVuEventRecord(rec *ddv1.VuEventRecord) *ddv1.VuEventRecord {
	if rec == nil {
		return nil
	}

	result := proto.Clone(rec).(*ddv1.VuEventRecord)

	// Anonymize timestamps
	result.SetBeginTime(opts.AnonymizeTimestamp(rec.GetBeginTime()))
	result.SetEndTime(opts.AnonymizeTimestamp(rec.GetEndTime()))

	// Anonymize card numbers
	result.SetCardNumberDriverSlotBegin(opts.AnonymizeFullCardNumber(rec.GetCardNumberDriverSlotBegin()))
	result.SetCardNumberCodriverSlotBegin(opts.AnonymizeFullCardNumber(rec.GetCardNumberCodriverSlotBegin()))
	result.SetCardNumberDriverSlotEnd(opts.AnonymizeFullCardNumber(rec.GetCardNumberDriverSlotEnd()))
	result.SetCardNumberCodriverSlotEnd(opts.AnonymizeFullCardNumber(rec.GetCardNumberCodriverSlotEnd()))

	// Anonymize similar card (if present)
	if rec.GetSimilarEventsNumber() > 0 {
		result.SetSimilarEventsNumber(rec.GetSimilarEventsNumber()) // Not PII
	}

	// Clear raw_data
	result.ClearRawData()

	return result
}
