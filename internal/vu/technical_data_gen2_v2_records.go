package vu

import (
	"github.com/way-platform/tachograph-go/internal/dd"
	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
)

func parseCardRecordArrayGen2V2(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V2_CardRecord, int, error) {
	recordsV1, bytesRead, err := parseCardRecordArrayGen2V1(data, offset)
	if err != nil {
		return nil, 0, err
	}
	records := make([]*vuv1.TechnicalDataGen2V2_CardRecord, len(recordsV1))
	for i, source := range recordsV1 {
		record := &vuv1.TechnicalDataGen2V2_CardRecord{}
		record.SetCardNumberAndGeneration(source.GetCardNumberAndGeneration())
		record.SetCardExtendedSerialNumber(source.GetCardExtendedSerialNumber())
		record.SetCardStructureVersion(source.GetCardStructureVersion())
		record.SetDriverIdentification(source.GetDriverIdentification())
		record.SetOwnerIdentification(source.GetOwnerIdentification())
		record.SetRawData(source.GetRawData())
		records[i] = record
	}
	return records, bytesRead, nil
}

func parseItsConsentRecordArrayGen2V2(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V2_ItsConsentRecord, int, error) {
	recordsV1, bytesRead, err := parseItsConsentRecordArrayGen2V1(data, offset)
	if err != nil {
		return nil, 0, err
	}
	records := make([]*vuv1.TechnicalDataGen2V2_ItsConsentRecord, len(recordsV1))
	for i, source := range recordsV1 {
		record := &vuv1.TechnicalDataGen2V2_ItsConsentRecord{}
		record.SetFullCardNumberAndGeneration(source.GetFullCardNumberAndGeneration())
		record.SetConsentStatus(source.GetConsentStatus())
		record.SetRawData(source.GetRawData())
		records[i] = record
	}
	return records, bytesRead, nil
}

func parsePowerSupplyInterruptionRecordArrayGen2V2(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V2_PowerSupplyInterruptionRecord, int, error) {
	recordsV1, bytesRead, err := parsePowerSupplyInterruptionRecordArrayGen2V1(data, offset)
	if err != nil {
		return nil, 0, err
	}
	records := make([]*vuv1.TechnicalDataGen2V2_PowerSupplyInterruptionRecord, len(recordsV1))
	for i, source := range recordsV1 {
		record := &vuv1.TechnicalDataGen2V2_PowerSupplyInterruptionRecord{}
		record.SetEventType(source.GetEventType())
		record.SetUnrecognizedEventType(source.GetUnrecognizedEventType())
		record.SetEventRecordPurpose(source.GetEventRecordPurpose())
		record.SetUnrecognizedEventRecordPurpose(source.GetUnrecognizedEventRecordPurpose())
		record.SetEventBeginTime(source.GetEventBeginTime())
		record.SetEventEndTime(source.GetEventEndTime())
		record.SetCardNumberAndGenerationDriverSlotBegin(source.GetCardNumberAndGenerationDriverSlotBegin())
		record.SetCardNumberAndGenerationDriverSlotEnd(source.GetCardNumberAndGenerationDriverSlotEnd())
		record.SetCardNumberAndGenerationCodriverSlotBegin(source.GetCardNumberAndGenerationCodriverSlotBegin())
		record.SetCardNumberAndGenerationCodriverSlotEnd(source.GetCardNumberAndGenerationCodriverSlotEnd())
		record.SetSimilarEventsNumber(source.GetSimilarEventsNumber())
		record.SetRawData(source.GetRawData())
		records[i] = record
	}
	return records, bytesRead, nil
}

func marshalCardRecordsGen2V2(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V2_CardRecord) ([]byte, error) {
	recordsV1 := make([]*vuv1.TechnicalDataGen2V1_CardRecord, len(records))
	for i, source := range records {
		record := &vuv1.TechnicalDataGen2V1_CardRecord{}
		record.SetCardNumberAndGeneration(source.GetCardNumberAndGeneration())
		record.SetCardExtendedSerialNumber(source.GetCardExtendedSerialNumber())
		record.SetCardStructureVersion(source.GetCardStructureVersion())
		record.SetDriverIdentification(source.GetDriverIdentification())
		record.SetOwnerIdentification(source.GetOwnerIdentification())
		record.SetRawData(source.GetRawData())
		recordsV1[i] = record
	}
	return marshalCardRecordsGen2V1(opts, recordsV1)
}

func marshalItsConsentRecordsGen2V2(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V2_ItsConsentRecord) ([]byte, error) {
	recordsV1 := make([]*vuv1.TechnicalDataGen2V1_ItsConsentRecord, len(records))
	for i, source := range records {
		record := &vuv1.TechnicalDataGen2V1_ItsConsentRecord{}
		record.SetFullCardNumberAndGeneration(source.GetFullCardNumberAndGeneration())
		record.SetConsentStatus(source.GetConsentStatus())
		record.SetRawData(source.GetRawData())
		recordsV1[i] = record
	}
	return marshalItsConsentRecordsGen2V1(opts, recordsV1)
}

func marshalPowerSupplyInterruptionRecordsGen2V2(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V2_PowerSupplyInterruptionRecord) ([]byte, error) {
	recordsV1 := make([]*vuv1.TechnicalDataGen2V1_PowerSupplyInterruptionRecord, len(records))
	for i, source := range records {
		record := &vuv1.TechnicalDataGen2V1_PowerSupplyInterruptionRecord{}
		record.SetEventType(source.GetEventType())
		record.SetUnrecognizedEventType(source.GetUnrecognizedEventType())
		record.SetEventRecordPurpose(source.GetEventRecordPurpose())
		record.SetUnrecognizedEventRecordPurpose(source.GetUnrecognizedEventRecordPurpose())
		record.SetEventBeginTime(source.GetEventBeginTime())
		record.SetEventEndTime(source.GetEventEndTime())
		record.SetCardNumberAndGenerationDriverSlotBegin(source.GetCardNumberAndGenerationDriverSlotBegin())
		record.SetCardNumberAndGenerationDriverSlotEnd(source.GetCardNumberAndGenerationDriverSlotEnd())
		record.SetCardNumberAndGenerationCodriverSlotBegin(source.GetCardNumberAndGenerationCodriverSlotBegin())
		record.SetCardNumberAndGenerationCodriverSlotEnd(source.GetCardNumberAndGenerationCodriverSlotEnd())
		record.SetSimilarEventsNumber(source.GetSimilarEventsNumber())
		record.SetRawData(source.GetRawData())
		recordsV1[i] = record
	}
	return marshalPowerSupplyInterruptionRecordsGen2V1(opts, recordsV1)
}
