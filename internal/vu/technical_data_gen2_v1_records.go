package vu

import (
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func parseSensorExternalGNSSCoupledRecordArrayGen2V1(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V1_CoupledGnss, int, error) {
	recordType, recordSize, noOfRecords, headerSize, err := parseRecordArrayHeader(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if recordType != recordTypeSensorExternalGNSSCoupled {
		return nil, 0, fmt.Errorf("expected SensorExternalGNSSCoupledRecord type 0x%02x, got 0x%02x", recordTypeSensorExternalGNSSCoupled, recordType)
	}
	if recordSize != 28 {
		return nil, 0, fmt.Errorf("expected CoupledGnss record size 28, got %d", recordSize)
	}

	var opts dd.UnmarshalOptions
	records := make([]*vuv1.TechnicalDataGen2V1_CoupledGnss, 0, noOfRecords)
	recordStart := offset + headerSize
	for i := range noOfRecords {
		recordEnd := recordStart + int(recordSize)
		if recordEnd > len(data) {
			return nil, 0, fmt.Errorf("insufficient data for CoupledGnss record %d", i)
		}
		sensor, err := opts.UnmarshalSensorPaired(data[recordStart:recordEnd])
		if err != nil {
			return nil, 0, fmt.Errorf("CoupledGnss record %d: %w", i, err)
		}
		record := &vuv1.TechnicalDataGen2V1_CoupledGnss{}
		record.SetSerialNumber(sensor.GetSerialNumber())
		record.SetApprovalNumber(sensor.GetApprovalNumber())
		record.SetCouplingDate(sensor.GetPairingDate())
		records = append(records, record)
		recordStart = recordEnd
	}
	return records, headerSize + int(recordSize)*int(noOfRecords), nil
}

func parseCardRecordArrayGen2V1(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V1_CardRecord, int, error) {
	recordType, recordSize, noOfRecords, headerSize, err := parseRecordArrayHeader(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if recordType != recordTypeVuCardRecord {
		return nil, 0, fmt.Errorf("expected VuCardRecord type 0x%02x, got 0x%02x", recordTypeVuCardRecord, recordType)
	}
	if recordSize != 45 {
		return nil, 0, fmt.Errorf("expected VuCardRecord size 45, got %d", recordSize)
	}

	var opts dd.UnmarshalOptions
	records := make([]*vuv1.TechnicalDataGen2V1_CardRecord, 0, noOfRecords)
	recordStart := offset + headerSize
	for i := range noOfRecords {
		recordEnd := recordStart + int(recordSize)
		if recordEnd > len(data) {
			return nil, 0, fmt.Errorf("insufficient data for VuCardRecord %d", i)
		}
		raw := data[recordStart:recordEnd]
		cardNumber, err := opts.UnmarshalFullCardNumberAndGeneration(raw[0:19])
		if err != nil {
			return nil, 0, fmt.Errorf("VuCardRecord %d card number and generation: %w", i, err)
		}
		serialNumber, err := opts.UnmarshalExtendedSerialNumber(raw[19:27])
		if err != nil {
			return nil, 0, fmt.Errorf("VuCardRecord %d extended serial number: %w", i, err)
		}
		structureVersion, err := opts.UnmarshalCardStructureVersion(raw[27:29])
		if err != nil {
			return nil, 0, fmt.Errorf("VuCardRecord %d card structure version: %w", i, err)
		}

		record := &vuv1.TechnicalDataGen2V1_CardRecord{}
		record.SetCardNumberAndGeneration(cardNumber)
		record.SetCardExtendedSerialNumber(serialNumber)
		record.SetCardStructureVersion(structureVersion)
		record.SetRawData(raw)
		switch cardNumber.GetFullCardNumber().GetCardType() {
		case ddv1.EquipmentType_DRIVER_CARD:
			identification, err := opts.UnmarshalDriverIdentification(raw[29:45])
			if err != nil {
				return nil, 0, fmt.Errorf("VuCardRecord %d driver identification: %w", i, err)
			}
			record.SetDriverIdentification(identification)
		case ddv1.EquipmentType_WORKSHOP_CARD, ddv1.EquipmentType_CONTROL_CARD, ddv1.EquipmentType_COMPANY_CARD:
			identification, err := opts.UnmarshalOwnerIdentification(raw[29:45])
			if err != nil {
				return nil, 0, fmt.Errorf("VuCardRecord %d owner identification: %w", i, err)
			}
			record.SetOwnerIdentification(identification)
		}
		records = append(records, record)
		recordStart = recordEnd
	}
	return records, headerSize + int(recordSize)*int(noOfRecords), nil
}

func parseItsConsentRecordArrayGen2V1(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V1_ItsConsentRecord, int, error) {
	recordType, recordSize, noOfRecords, headerSize, err := parseRecordArrayHeader(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if recordType != recordTypeVuITSConsentRecord {
		return nil, 0, fmt.Errorf("expected VuITSConsentRecord type 0x%02x, got 0x%02x", recordTypeVuITSConsentRecord, recordType)
	}
	if recordSize != 20 {
		return nil, 0, fmt.Errorf("expected VuITSConsentRecord size 20, got %d", recordSize)
	}

	var opts dd.UnmarshalOptions
	records := make([]*vuv1.TechnicalDataGen2V1_ItsConsentRecord, 0, noOfRecords)
	recordStart := offset + headerSize
	for i := range noOfRecords {
		recordEnd := recordStart + int(recordSize)
		if recordEnd > len(data) {
			return nil, 0, fmt.Errorf("insufficient data for VuITSConsentRecord %d", i)
		}
		raw := data[recordStart:recordEnd]
		cardNumber, err := opts.UnmarshalFullCardNumberAndGeneration(raw[0:19])
		if err != nil {
			return nil, 0, fmt.Errorf("VuITSConsentRecord %d card number: %w", i, err)
		}
		record := &vuv1.TechnicalDataGen2V1_ItsConsentRecord{}
		record.SetFullCardNumberAndGeneration(cardNumber)
		record.SetConsentStatus(raw[19] != 0)
		record.SetRawData(raw)
		records = append(records, record)
		recordStart = recordEnd
	}
	return records, headerSize + int(recordSize)*int(noOfRecords), nil
}

func parsePowerSupplyInterruptionRecordArrayGen2V1(data []byte, offset int) ([]*vuv1.TechnicalDataGen2V1_PowerSupplyInterruptionRecord, int, error) {
	recordType, recordSize, noOfRecords, headerSize, err := parseRecordArrayHeader(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if recordType != recordTypeVuPowerSupplyInterruption {
		return nil, 0, fmt.Errorf("expected VuPowerSupplyInterruptionRecord type 0x%02x, got 0x%02x", recordTypeVuPowerSupplyInterruption, recordType)
	}
	if recordSize != 87 {
		return nil, 0, fmt.Errorf("expected VuPowerSupplyInterruptionRecord size 87, got %d", recordSize)
	}

	var opts dd.UnmarshalOptions
	records := make([]*vuv1.TechnicalDataGen2V1_PowerSupplyInterruptionRecord, 0, noOfRecords)
	recordStart := offset + headerSize
	for i := range noOfRecords {
		recordEnd := recordStart + int(recordSize)
		if recordEnd > len(data) {
			return nil, 0, fmt.Errorf("insufficient data for VuPowerSupplyInterruptionRecord %d", i)
		}
		raw := data[recordStart:recordEnd]
		record := &vuv1.TechnicalDataGen2V1_PowerSupplyInterruptionRecord{}
		if eventType, err := dd.UnmarshalEnum[ddv1.EventFaultType](raw[0]); err != nil {
			record.SetUnrecognizedEventType(int32(raw[0]))
		} else {
			record.SetEventType(eventType)
		}
		if purpose, err := dd.UnmarshalEnum[ddv1.EventFaultRecordPurpose](raw[1]); err != nil {
			record.SetUnrecognizedEventRecordPurpose(int32(raw[1]))
		} else {
			record.SetEventRecordPurpose(purpose)
		}
		begin, err := opts.UnmarshalTimeReal(raw[2:6])
		if err != nil {
			return nil, 0, fmt.Errorf("VuPowerSupplyInterruptionRecord %d begin time: %w", i, err)
		}
		end, err := opts.UnmarshalTimeReal(raw[6:10])
		if err != nil {
			return nil, 0, fmt.Errorf("VuPowerSupplyInterruptionRecord %d end time: %w", i, err)
		}
		record.SetEventBeginTime(begin)
		record.SetEventEndTime(end)
		cardSetters := []func(*ddv1.FullCardNumberAndGeneration){
			record.SetCardNumberAndGenerationDriverSlotBegin,
			record.SetCardNumberAndGenerationDriverSlotEnd,
			record.SetCardNumberAndGenerationCodriverSlotBegin,
			record.SetCardNumberAndGenerationCodriverSlotEnd,
		}
		for cardIndex, setCard := range cardSetters {
			start := 10 + cardIndex*19
			card, err := opts.UnmarshalFullCardNumberAndGeneration(raw[start : start+19])
			if err != nil {
				return nil, 0, fmt.Errorf("VuPowerSupplyInterruptionRecord %d card %d: %w", i, cardIndex, err)
			}
			setCard(card)
		}
		record.SetSimilarEventsNumber(int32(raw[86]))
		record.SetRawData(raw)
		records = append(records, record)
		recordStart = recordEnd
	}
	return records, headerSize + int(recordSize)*int(noOfRecords), nil
}

func marshalSensorExternalGNSSCoupledRecordsGen2V1(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V1_CoupledGnss) ([]byte, error) {
	result := make([]byte, 0, len(records)*28)
	for i, record := range records {
		sensor := &ddv1.SensorPaired{}
		sensor.SetSerialNumber(record.GetSerialNumber())
		sensor.SetApprovalNumber(record.GetApprovalNumber())
		sensor.SetPairingDate(record.GetCouplingDate())
		raw, err := opts.MarshalSensorPaired(sensor)
		if err != nil {
			return nil, fmt.Errorf("CoupledGnss record %d: %w", i, err)
		}
		result = append(result, raw...)
	}
	return result, nil
}

func marshalCardRecordsGen2V1(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V1_CardRecord) ([]byte, error) {
	result := make([]byte, 0, len(records)*45)
	for i, record := range records {
		if len(record.GetRawData()) == 45 {
			result = append(result, record.GetRawData()...)
			continue
		}
		cardNumber, err := opts.MarshalFullCardNumberAndGeneration(record.GetCardNumberAndGeneration())
		if err != nil {
			return nil, fmt.Errorf("VuCardRecord %d card number: %w", i, err)
		}
		serialNumber, err := opts.MarshalExtendedSerialNumber(record.GetCardExtendedSerialNumber())
		if err != nil {
			return nil, fmt.Errorf("VuCardRecord %d serial number: %w", i, err)
		}
		structureVersion, err := opts.MarshalCardStructureVersion(record.GetCardStructureVersion())
		if err != nil {
			return nil, fmt.Errorf("VuCardRecord %d structure version: %w", i, err)
		}
		result = append(result, cardNumber...)
		result = append(result, serialNumber...)
		result = append(result, structureVersion...)
		switch record.GetCardNumberAndGeneration().GetFullCardNumber().GetCardType() {
		case ddv1.EquipmentType_DRIVER_CARD:
			identification, err := opts.MarshalDriverIdentification(record.GetDriverIdentification())
			if err != nil {
				return nil, fmt.Errorf("VuCardRecord %d driver identification: %w", i, err)
			}
			result = append(result, identification...)
		default:
			identification, err := opts.MarshalOwnerIdentification(record.GetOwnerIdentification())
			if err != nil {
				return nil, fmt.Errorf("VuCardRecord %d owner identification: %w", i, err)
			}
			result = append(result, identification...)
		}
	}
	return result, nil
}

func marshalItsConsentRecordsGen2V1(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V1_ItsConsentRecord) ([]byte, error) {
	result := make([]byte, 0, len(records)*20)
	for i, record := range records {
		if len(record.GetRawData()) == 20 {
			result = append(result, record.GetRawData()...)
			continue
		}
		cardNumber, err := opts.MarshalFullCardNumberAndGeneration(record.GetFullCardNumberAndGeneration())
		if err != nil {
			return nil, fmt.Errorf("VuITSConsentRecord %d card number: %w", i, err)
		}
		result = append(result, cardNumber...)
		if record.GetConsentStatus() {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
	}
	return result, nil
}

func marshalPowerSupplyInterruptionRecordsGen2V1(opts dd.MarshalOptions, records []*vuv1.TechnicalDataGen2V1_PowerSupplyInterruptionRecord) ([]byte, error) {
	result := make([]byte, 0, len(records)*87)
	for i, record := range records {
		if len(record.GetRawData()) == 87 {
			result = append(result, record.GetRawData()...)
			continue
		}
		eventType, err := marshalRecognizedOrRawEnum(record.GetEventType(), record.GetUnrecognizedEventType())
		if err != nil {
			return nil, fmt.Errorf("VuPowerSupplyInterruptionRecord %d event type: %w", i, err)
		}
		purpose, err := marshalRecognizedOrRawEnum(record.GetEventRecordPurpose(), record.GetUnrecognizedEventRecordPurpose())
		if err != nil {
			return nil, fmt.Errorf("VuPowerSupplyInterruptionRecord %d purpose: %w", i, err)
		}
		result = append(result, eventType, purpose)
		begin, err := opts.MarshalTimeReal(record.GetEventBeginTime())
		if err != nil {
			return nil, fmt.Errorf("VuPowerSupplyInterruptionRecord %d begin time: %w", i, err)
		}
		end, err := opts.MarshalTimeReal(record.GetEventEndTime())
		if err != nil {
			return nil, fmt.Errorf("VuPowerSupplyInterruptionRecord %d end time: %w", i, err)
		}
		result = append(result, begin...)
		result = append(result, end...)
		cards := []*ddv1.FullCardNumberAndGeneration{
			record.GetCardNumberAndGenerationDriverSlotBegin(),
			record.GetCardNumberAndGenerationDriverSlotEnd(),
			record.GetCardNumberAndGenerationCodriverSlotBegin(),
			record.GetCardNumberAndGenerationCodriverSlotEnd(),
		}
		for cardIndex, card := range cards {
			raw, err := opts.MarshalFullCardNumberAndGeneration(card)
			if err != nil {
				return nil, fmt.Errorf("VuPowerSupplyInterruptionRecord %d card %d: %w", i, cardIndex, err)
			}
			result = append(result, raw...)
		}
		result = append(result, byte(record.GetSimilarEventsNumber()))
	}
	return result, nil
}

func marshalRecognizedOrRawEnum[T interface {
	~int32
	protoreflect.Enum
}](value T, unrecognized int32) (byte, error) {
	if unrecognized != 0 {
		return byte(unrecognized), nil
	}
	return dd.MarshalEnum(value)
}
