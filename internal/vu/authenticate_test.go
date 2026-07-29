package vu

import (
	"encoding/binary"
	"testing"

	vuv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/vu/v1"
)

func TestUnwrapGen2SignatureRecordArray(t *testing.T) {
	signature := make([]byte, 64)
	for i := range signature {
		signature[i] = byte(i)
	}
	recordArray := appendRecordArrayHeader(nil, recordTypeSignature, uint16(len(signature)), 1)
	recordArray = append(recordArray, signature...)

	got, err := unwrapGen2SignatureRecordArray(recordArray)
	if err != nil {
		t.Fatalf("unwrapGen2SignatureRecordArray() error = %v", err)
	}
	if len(got) != len(signature) {
		t.Fatalf("signature length = %d, want %d", len(got), len(signature))
	}
	for i := range signature {
		if got[i] != signature[i] {
			t.Fatalf("signature byte %d = %d, want %d", i, got[i], signature[i])
		}
	}
}

func TestUnwrapGen2SignatureRecordArrayRejectsMalformedWrappers(t *testing.T) {
	valid := appendRecordArrayHeader(nil, recordTypeSignature, 64, 1)
	valid = append(valid, make([]byte, 64)...)

	tests := map[string]func([]byte){
		"wrong record type":  func(data []byte) { data[0] = 0x09 },
		"wrong record count": func(data []byte) { binary.BigEndian.PutUint16(data[3:5], 2) },
		"wrong record size":  func(data []byte) { binary.BigEndian.PutUint16(data[1:3], 63) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			data := append([]byte(nil), valid...)
			mutate(data)
			if _, err := unwrapGen2SignatureRecordArray(data); err == nil {
				t.Fatal("unwrapGen2SignatureRecordArray() error = nil, want error")
			}
		})
	}
}

func TestSignedDataForGen2OverviewExcludesCertificateArrays(t *testing.T) {
	var data []byte
	data = appendRecordArrayHeader(data, 0x04, 3, 1)
	data = append(data, 1, 2, 3)
	data = appendRecordArrayHeader(data, 0x0f, 2, 1)
	data = append(data, 4, 5)
	signedStart := len(data)
	data = appendRecordArrayHeader(data, 0x0a, 1, 1)
	data = append(data, 6)

	got, err := signedDataForGen2Record(vuv1.TransferType_OVERVIEW_GEN2_V1, data)
	if err != nil {
		t.Fatalf("signedDataForGen2Record() error = %v", err)
	}
	if len(got) != len(data)-signedStart || got[len(got)-1] != 6 {
		t.Fatalf("signed data = %x, want %x", got, data[signedStart:])
	}
}
