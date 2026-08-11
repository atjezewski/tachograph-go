package card

import (
	"testing"

	"github.com/way-platform/tachograph-go/internal/security"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
)

func TestGen1CardCertificateRole(t *testing.T) {
	tests := []struct {
		cardType cardv1.CardType
		want     security.RsaCertificateRole
	}{
		{cardv1.CardType_DRIVER_CARD, security.RsaCertificateRoleDriverCard},
		{cardv1.CardType_WORKSHOP_CARD, security.RsaCertificateRoleWorkshopCard},
		{cardv1.CardType_CONTROL_CARD, security.RsaCertificateRoleControlCard},
		{cardv1.CardType_COMPANY_CARD, security.RsaCertificateRoleCompanyCard},
	}
	for _, tt := range tests {
		got, err := gen1CardCertificateRole(tt.cardType)
		if err != nil {
			t.Fatalf("gen1CardCertificateRole(%v) error = %v", tt.cardType, err)
		}
		if got != tt.want {
			t.Errorf("gen1CardCertificateRole(%v) = %v, want %v", tt.cardType, got, tt.want)
		}
	}
}

func TestGen2CardCertificateRole(t *testing.T) {
	tests := []struct {
		cardType cardv1.CardType
		want     security.EccCertificateRole
	}{
		{cardv1.CardType_DRIVER_CARD, security.EccCertificateRoleDriverCardSign},
		{cardv1.CardType_WORKSHOP_CARD, security.EccCertificateRoleWorkshopCardSign},
	}
	for _, tt := range tests {
		got, err := gen2CardCertificateRole(tt.cardType)
		if err != nil {
			t.Fatalf("gen2CardCertificateRole(%v) error = %v", tt.cardType, err)
		}
		if got != tt.want {
			t.Errorf("gen2CardCertificateRole(%v) = %v, want %v", tt.cardType, got, tt.want)
		}
	}
	if _, err := gen2CardCertificateRole(cardv1.CardType_CONTROL_CARD); err == nil {
		t.Fatal("gen2CardCertificateRole(CONTROL_CARD) accepted a card type without Card_Sign")
	}
}
