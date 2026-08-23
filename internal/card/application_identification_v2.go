package card

import (
	"encoding/binary"
	"fmt"

	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
)

// unmarshalApplicationIdentificationV2 parses the binary data for an EF_ApplicationIdentificationV2 record.
//
// The data type `DriverCardApplicationIdentificationV2` is specified in the Data Dictionary, Section 2.61a.
//
// ASN.1 Definition:
//
//	DriverCardApplicationIdentificationV2 ::= SEQUENCE {
//	    lengthOfFollowingData        LengthOfFollowingData,
//	    noOfBorderCrossingRecords    NoOfBorderCrossingRecords,
//	    noOfLoadUnloadRecords        NoOfLoadUnloadRecords,
//	    noOfLoadTypeEntryRecords     NoOfLoadTypeEntryRecords,
//	    vuConfigurationLengthRange   VuConfigurationLengthRange
//	}
//
// Binary Layout (fixed length, 10 bytes): every component is a two-byte
// INTEGER(0..2^16-1). The record counts are the sizes of the ring buffers in
// EF_Border_Crossings, EF_Load_Unload_Operations and EF_Load_Type_Entries, so
// they do not fit in a byte: a driver card holds 1120, 1624 and 336 records
// respectively.
//
//   - Bytes 0-1: lengthOfFollowingData
//   - Bytes 2-3: noOfBorderCrossingRecords
//   - Bytes 4-5: noOfLoadUnloadRecords
//   - Bytes 6-7: noOfLoadTypeEntryRecords
//   - Bytes 8-9: vuConfigurationLengthRange
func (opts UnmarshalOptions) unmarshalApplicationIdentificationV2(data []byte) (*cardv1.ApplicationIdentificationV2, error) {
	const (
		idxLengthOfFollowingData       = 0
		idxBorderCrossingRecords       = 2
		idxLoadUnloadRecords           = 4
		idxLoadTypeEntryRecords        = 6
		idxVuConfigurationLengthRange  = 8
		lenApplicationIdentificationV2 = 10
	)

	if len(data) != lenApplicationIdentificationV2 {
		return nil, fmt.Errorf(
			"invalid data length for DriverCardApplicationIdentificationV2: got %d bytes, want %d",
			len(data), lenApplicationIdentificationV2,
		)
	}

	var target cardv1.ApplicationIdentificationV2

	driver := &cardv1.ApplicationIdentificationV2_Driver{}
	driver.SetLengthOfFollowingData(int32(binary.BigEndian.Uint16(data[idxLengthOfFollowingData:])))
	driver.SetBorderCrossingRecordsCount(int32(binary.BigEndian.Uint16(data[idxBorderCrossingRecords:])))
	driver.SetLoadUnloadRecordsCount(int32(binary.BigEndian.Uint16(data[idxLoadUnloadRecords:])))
	driver.SetLoadTypeEntryRecordsCount(int32(binary.BigEndian.Uint16(data[idxLoadTypeEntryRecords:])))
	driver.SetVuConfigurationLengthRange(int32(binary.BigEndian.Uint16(data[idxVuConfigurationLengthRange:])))

	target.SetDriver(driver)
	target.SetCardType(cardv1.CardType_DRIVER_CARD)

	return &target, nil
}

// MarshalCardApplicationIdentificationV2 marshals application identification V2 data.
//
// The data type `DriverCardApplicationIdentificationV2` is specified in the Data Dictionary, Section 2.61a.
// Driver and workshop cards carry five two-byte components; company and control
// cards carry only the length and the VU configuration length range.
func (opts MarshalOptions) MarshalCardApplicationIdentificationV2(appIdV2 *cardv1.ApplicationIdentificationV2) ([]byte, error) {
	if appIdV2 == nil {
		return nil, nil
	}

	switch appIdV2.GetCardType() {
	case cardv1.CardType_DRIVER_CARD:
		driver := appIdV2.GetDriver()
		return appendRecordCounts(driver.GetLengthOfFollowingData(),
			driver.GetBorderCrossingRecordsCount(),
			driver.GetLoadUnloadRecordsCount(),
			driver.GetLoadTypeEntryRecordsCount(),
			driver.GetVuConfigurationLengthRange()), nil
	case cardv1.CardType_WORKSHOP_CARD:
		workshop := appIdV2.GetWorkshop()
		return appendRecordCounts(workshop.GetLengthOfFollowingData(),
			workshop.GetBorderCrossingRecordsCount(),
			workshop.GetLoadUnloadRecordsCount(),
			workshop.GetLoadTypeEntryRecordsCount(),
			workshop.GetVuConfigurationLengthRange()), nil
	case cardv1.CardType_COMPANY_CARD:
		company := appIdV2.GetCompany()
		return appendRecordCounts(company.GetLengthOfFollowingData(),
			company.GetVuConfigurationLengthRange()), nil
	case cardv1.CardType_CONTROL_CARD:
		control := appIdV2.GetControl()
		return appendRecordCounts(control.GetLengthOfFollowingData(),
			control.GetVuConfigurationLengthRange()), nil
	default:
		return nil, fmt.Errorf("unsupported card type for ApplicationIdentificationV2: %v", appIdV2.GetCardType())
	}
}

// appendRecordCounts encodes the components of an ApplicationIdentificationV2
// record, each a two-byte INTEGER(0..2^16-1).
func appendRecordCounts(values ...int32) []byte {
	dst := make([]byte, 0, len(values)*2)
	for _, value := range values {
		dst = binary.BigEndian.AppendUint16(dst, uint16(value))
	}
	return dst
}
