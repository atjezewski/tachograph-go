package card

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/way-platform/tachograph-go/internal/dd"

	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

// unmarshalIcc parses the binary data for an EF_ICC record.
//
// The data type `CardIccIdentification` is specified in the Data Dictionary, Section 2.23.
//
// ASN.1 Definition:
//
//	CardIccIdentification ::= SEQUENCE {
//	    clockStop                   OCTET STRING (SIZE(1)),
//	    cardExtendedSerialNumber    ExtendedSerialNumber,    -- 8 bytes
//	    cardApprovalNumber          CardApprovalNumber,      -- 8 bytes
//	    cardPersonaliserID          ManufacturerCode,        -- 1 byte
//	    embedderIcAssemblerId       EmbedderIcAssemblerId,   -- 5 bytes
//	    icIdentifier                OCTET STRING (SIZE(2))
//	}
func (opts UnmarshalOptions) unmarshalIcc(data []byte) (*cardv1.Icc, error) {
	const (
		lenClockStop                = 1
		lenCardExtendedSerialNumber = 8
		lenCardApprovalNumber       = 8
		lenCardPersonaliserId       = 1
		lenEmbedderIcAssemblerId    = 5
		lenIcIdentifier             = 2
		lenCardIccIdentification    = lenClockStop + lenCardExtendedSerialNumber + lenCardApprovalNumber + lenCardPersonaliserId + lenEmbedderIcAssemblerId + lenIcIdentifier
	)

	var icc cardv1.Icc
	if len(data) < lenCardIccIdentification {
		return nil, errors.New("not enough data for IccIdentification")
	}
	offset := 0

	// Read clock stop (1 byte)
	if offset+1 > len(data) {
		return nil, fmt.Errorf("insufficient data for clock stop")
	}
	// Convert clock stop byte to ClockStopMode enum using generic helper
	if clockStopMode, err := dd.UnmarshalEnum[ddv1.ClockStopMode](data[offset]); err == nil {
		icc.SetClockStop(clockStopMode)
	} else {
		return nil, fmt.Errorf("invalid clock stop mode: %w", err)
	}
	offset++

	// Create ExtendedSerialNumber structure
	esn := &ddv1.ExtendedSerialNumber{}
	// Read the 8-byte extended serial number
	if offset+lenCardExtendedSerialNumber > len(data) {
		return nil, fmt.Errorf("insufficient data for card extended serial number")
	}
	serialBytes := data[offset : offset+lenCardExtendedSerialNumber]
	offset += lenCardExtendedSerialNumber
	if len(serialBytes) >= lenCardExtendedSerialNumber {
		// Parse the fields according to ExtendedSerialNumber structure
		// First 4 bytes: serial number (big-endian)
		serialNum := binary.BigEndian.Uint32(serialBytes[0:4])
		esn.SetSerialNumber(int64(serialNum))

		// Next 2 bytes: month/year BCD (MMYY format)
		if len(serialBytes) > 5 {
			monthYear, err := opts.UnmarshalMonthYear(serialBytes[4:6])
			if err != nil {
				return nil, fmt.Errorf("failed to parse month/year: %w", err)
			}
			esn.SetMonthYear(monthYear)
		}

		// Next byte: equipment type (convert from protocol value using generic helper)
		if len(serialBytes) > 6 {
			if equipmentType, err := dd.UnmarshalEnum[ddv1.EquipmentType](serialBytes[6]); err == nil {
				esn.SetType(equipmentType)
			} else {
				return nil, fmt.Errorf("invalid equipment type in extended serial number: %w", err)
			}
		}

		// Last byte: manufacturer code
		if len(serialBytes) > 7 {
			esn.SetManufacturerCode(int32(serialBytes[7]))
		}
	}
	icc.SetCardExtendedSerialNumber(esn)

	// Read card approval number (8 bytes)
	if offset+lenCardApprovalNumber > len(data) {
		return nil, fmt.Errorf("insufficient data for card approval number")
	}
	cardApprovalNumber, err := opts.UnmarshalIa5StringValue(data[offset : offset+lenCardApprovalNumber])
	if err != nil {
		return nil, fmt.Errorf("failed to read card approval number: %w", err)
	}
	icc.SetCardApprovalNumber(cardApprovalNumber)
	offset += lenCardApprovalNumber

	// Read card personaliser ID (1 byte)
	if offset+1 > len(data) {
		return nil, fmt.Errorf("insufficient data for card personaliser ID")
	}
	personaliser := data[offset]
	icc.SetCardPersonaliserId(int32(personaliser))
	offset++

	// Create EmbedderIcAssemblerId structure (5 bytes)
	if offset+lenEmbedderIcAssemblerId > len(data) {
		return nil, fmt.Errorf("insufficient data for embedder IC assembler ID")
	}
	embedder := data[offset : offset+lenEmbedderIcAssemblerId]
	offset += lenEmbedderIcAssemblerId
	eia := &cardv1.Icc_EmbedderIcAssemblerId{}
	if len(embedder) >= lenEmbedderIcAssemblerId {
		// Country code (2 bytes, IA5String)
		countryCode, err := opts.UnmarshalIa5StringValue(embedder[0:2])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal country code: %w", err)
		}
		eia.SetCountryCode(countryCode)

		// Preserve the exact bytes. Keep populating the original IA5String field
		// as a compatibility view for callers compiled against the public API.
		eia.SetModuleEmbedderRaw(bytes.Clone(embedder[2:4]))
		moduleEmbedder, err := opts.UnmarshalIa5StringValue(embedder[2:4])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal module embedder: %w", err)
		}
		eia.SetModuleEmbedder(moduleEmbedder)

		// Manufacturer information (1 byte)
		eia.SetManufacturerInformation(int32(embedder[4]))
	}
	icc.SetEmbedderIcAssemblerId(eia)

	// Read IC identifier (2 bytes)
	if offset+lenIcIdentifier > len(data) {
		return nil, fmt.Errorf("insufficient data for IC identifier")
	}
	icIdentifier := data[offset : offset+lenIcIdentifier]
	// offset += lenIcIdentifier // Not needed as this is the last field
	icc.SetIcIdentifier(icIdentifier)
	return &icc, nil
}

// MarshalIcc marshals the binary representation of an EF_ICC message to bytes.
//
// The data type `CardIccIdentification` is specified in the Data Dictionary, Section 2.23.
//
// ASN.1 Definition:
//
//	CardIccIdentification ::= SEQUENCE {
//	    clockStop                   OCTET STRING (SIZE(1)),
//	    cardExtendedSerialNumber    ExtendedSerialNumber,    -- 8 bytes
//	    cardApprovalNumber          CardApprovalNumber,      -- 8 bytes
//	    cardPersonaliserID          ManufacturerCode,        -- 1 byte
//	    embedderIcAssemblerId       EmbedderIcAssemblerId,   -- 5 bytes
//	    icIdentifier                OCTET STRING (SIZE(2))
//	}
func (opts MarshalOptions) MarshalIcc(icc *cardv1.Icc) ([]byte, error) {
	const (
		lenClockStop                = 1
		lenCardExtendedSerialNumber = 8
		lenCardApprovalNumber       = 8
		lenCardPersonaliserId       = 1
		lenEmbedderIcAssemblerId    = 5
		lenIcIdentifier             = 2
	)

	var dst []byte

	// Append clock stop (1 byte)
	clockStopByte, err := dd.MarshalEnum(icc.GetClockStop())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal clock stop: %w", err)
	}
	dst = append(dst, clockStopByte)

	// Append extended serial number (8 bytes)
	esnBytes, err := opts.MarshalExtendedSerialNumberAsString(icc.GetCardExtendedSerialNumber(), lenCardExtendedSerialNumber)
	if err != nil {
		return nil, err
	}
	dst = append(dst, esnBytes...)

	// Append card approval number (8 bytes)
	approvalBytes, err := opts.MarshalIa5StringValue(icc.GetCardApprovalNumber())
	if err != nil {
		return nil, err
	}
	dst = append(dst, approvalBytes...)

	// Append card personaliser ID (1 byte)
	dst = append(dst, byte(icc.GetCardPersonaliserId()))

	// Append embedder IC assembler ID (5 bytes)
	eiaBytes, err := opts.MarshalEmbedderIcAssemblerId(icc.GetEmbedderIcAssemblerId())
	if err != nil {
		return nil, err
	}
	dst = append(dst, eiaBytes...)

	// Append IC identifier (2 bytes)
	dst = append(dst, icc.GetIcIdentifier()...)
	return dst, nil
}

// MarshalEmbedderIcAssemblerId marshals an EmbedderIcAssemblerId structure (5 bytes total)
func (opts MarshalOptions) MarshalEmbedderIcAssemblerId(eia *cardv1.Icc_EmbedderIcAssemblerId) ([]byte, error) {
	const (
		lenEmbedderIcAssemblerId = 5
		lenModuleEmbedder        = 2
	)

	if eia == nil {
		// Return default values: 5 zero bytes
		return make([]byte, lenEmbedderIcAssemblerId), nil
	}

	var dst []byte

	// Append country code (2 bytes, IA5String)
	countryCodeBytes, err := opts.MarshalIa5StringValue(eia.GetCountryCode())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal country code: %w", err)
	}
	dst = append(dst, countryCodeBytes...)

	// Prefer the exact bytes when available, with the original IA5String field
	// retained as a compatibility fallback.
	if raw := eia.GetModuleEmbedderRaw(); len(raw) > 0 {
		if len(raw) != lenModuleEmbedder {
			return nil, fmt.Errorf("module embedder raw length must be %d bytes, got %d", lenModuleEmbedder, len(raw))
		}
		dst = append(dst, raw...)
	} else {
		moduleEmbedder, err := opts.MarshalIa5StringValue(eia.GetModuleEmbedder())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal module embedder: %w", err)
		}
		dst = append(dst, moduleEmbedder...)
	}

	// Append manufacturer information (1 byte)
	dst = append(dst, byte(eia.GetManufacturerInformation()))

	return dst, nil
}

// anonymizeIcc creates an anonymized copy of Icc,
// replacing sensitive information with static, deterministic test values.
func (opts AnonymizeOptions) anonymizeIcc(icc *cardv1.Icc) *cardv1.Icc {
	if icc == nil {
		return nil
	}

	anonymized := &cardv1.Icc{}

	// Create DD anonymize options
	ddOpts := dd.AnonymizeOptions{
		PreserveDistanceAndTrips: opts.PreserveDistanceAndTrips,
		PreserveTimestamps:       opts.PreserveTimestamps,
	}

	// Preserve clock stop mode (not sensitive)
	anonymized.SetClockStop(icc.GetClockStop())

	// Anonymize extended serial number
	if esn := icc.GetCardExtendedSerialNumber(); esn != nil {
		anonymizedESN := &ddv1.ExtendedSerialNumber{}

		// Use static test serial number
		anonymizedESN.SetSerialNumber(12345678)

		// Use static test date: January 2020 (month=1, year=2020)
		monthYear := &ddv1.MonthYear{}
		monthYear.SetYear(2020)
		monthYear.SetMonth(1)
		anonymizedESN.SetMonthYear(monthYear)

		// Preserve equipment type (not sensitive, categorical data)
		anonymizedESN.SetType(esn.GetType())

		// Use static test manufacturer code
		anonymizedESN.SetManufacturerCode(0x99)

		anonymized.SetCardExtendedSerialNumber(anonymizedESN)
	}

	// Anonymize card approval number
	if icc.GetCardApprovalNumber() != nil {
		anonymized.SetCardApprovalNumber(ddOpts.AnonymizeIa5StringValue(icc.GetCardApprovalNumber()))
	}

	// Use static test personaliser ID
	anonymized.SetCardPersonaliserId(0xAA)

	// Anonymize embedder IC assembler ID
	if eia := icc.GetEmbedderIcAssemblerId(); eia != nil {
		anonymizedEIA := &cardv1.Icc_EmbedderIcAssemblerId{}

		// Anonymize country code
		if eia.GetCountryCode() != nil {
			anonymizedEIA.SetCountryCode(ddOpts.AnonymizeIa5StringValue(eia.GetCountryCode()))
		}

		// Keep the compatibility and exact-byte views consistent.
		anonymizedEIA.SetModuleEmbedderRaw([]byte{'*', '*'})
		if eia.GetModuleEmbedder() != nil {
			anonymizedEIA.SetModuleEmbedder(ddOpts.AnonymizeIa5StringValue(eia.GetModuleEmbedder()))
		}

		// Use static test manufacturer information
		anonymizedEIA.SetManufacturerInformation(0xBB)

		anonymized.SetEmbedderIcAssemblerId(anonymizedEIA)
	}

	// Use static test IC identifier
	anonymized.SetIcIdentifier([]byte{0xCC, 0xDD})

	return anonymized
}
