package vu

import (
	"testing"

	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

func TestParseTechnicalDataGen2V1AdditionalRecordArrays(t *testing.T) {
	t.Run("card", func(t *testing.T) {
		raw := make([]byte, 45)
		for i := 0; i < 19; i++ {
			raw[i] = 0xff
		}
		data := appendRecordArrayHeader(nil, recordTypeVuCardRecord, 45, 1)
		data = append(data, raw...)

		records, consumed, err := parseCardRecordArrayGen2V1(data, 0)
		if err != nil {
			t.Fatalf("parseCardRecordArrayGen2V1() error = %v", err)
		}
		if consumed != len(data) || len(records) != 1 {
			t.Fatalf("got consumed=%d records=%d, want consumed=%d records=1", consumed, len(records), len(data))
		}
		if len(records[0].GetRawData()) != 45 {
			t.Fatalf("raw_data length = %d, want 45", len(records[0].GetRawData()))
		}
	})

	t.Run("ITS consent", func(t *testing.T) {
		raw := make([]byte, 20)
		for i := 0; i < 19; i++ {
			raw[i] = 0xff
		}
		raw[19] = 1
		data := appendRecordArrayHeader(nil, recordTypeVuITSConsentRecord, 20, 1)
		data = append(data, raw...)

		records, consumed, err := parseItsConsentRecordArrayGen2V1(data, 0)
		if err != nil {
			t.Fatalf("parseItsConsentRecordArrayGen2V1() error = %v", err)
		}
		if consumed != len(data) || len(records) != 1 || !records[0].GetConsentStatus() {
			t.Fatalf("got consumed=%d records=%d consent=%v", consumed, len(records), records[0].GetConsentStatus())
		}
	})

	t.Run("power interruption", func(t *testing.T) {
		raw := make([]byte, 87)
		raw[0] = 0x08
		raw[1] = 0x00
		for cardOffset := 10; cardOffset < 86; cardOffset++ {
			raw[cardOffset] = 0xff
		}
		raw[86] = 3
		data := appendRecordArrayHeader(nil, recordTypeVuPowerSupplyInterruption, 87, 1)
		data = append(data, raw...)

		records, consumed, err := parsePowerSupplyInterruptionRecordArrayGen2V1(data, 0)
		if err != nil {
			t.Fatalf("parsePowerSupplyInterruptionRecordArrayGen2V1() error = %v", err)
		}
		if consumed != len(data) || len(records) != 1 {
			t.Fatalf("got consumed=%d records=%d, want consumed=%d records=1", consumed, len(records), len(data))
		}
		if records[0].GetEventType() != ddv1.EventFaultType_GENERAL_POWER_SUPPLY_INTERRUPTION {
			t.Fatalf("event_type = %v, want GENERAL_POWER_SUPPLY_INTERRUPTION", records[0].GetEventType())
		}
		if records[0].GetSimilarEventsNumber() != 3 {
			t.Fatalf("similar_events_number = %d, want 3", records[0].GetSimilarEventsNumber())
		}
	})
}

func TestSizeOfTechnicalDataGen2V1ScansUntilSignature(t *testing.T) {
	var data []byte
	for _, recordType := range []byte{
		recordTypeVuIdentification,
		recordTypeSensorPairedRecord,
		recordTypeSensorExternalGNSSCoupled,
		recordTypeVuCalibrationRecord,
		recordTypeVuCardRecord,
		recordTypeVuITSConsentRecord,
		recordTypeVuPowerSupplyInterruption,
	} {
		data = appendRecordArrayHeader(data, recordType, 1, 0)
	}
	data = appendRecordArrayHeader(data, recordTypeSignature, 64, 1)
	data = append(data, make([]byte, 64)...)

	totalSize, signatureSize, err := sizeOfTechnicalDataGen2V1(data)
	if err != nil {
		t.Fatalf("sizeOfTechnicalDataGen2V1() error = %v", err)
	}
	if totalSize != len(data) {
		t.Fatalf("totalSize = %d, want %d", totalSize, len(data))
	}
	if signatureSize != 69 {
		t.Fatalf("signatureSize = %d, want 69", signatureSize)
	}
}
