package tachograph

import (
	"context"
	"fmt"
	"time"

	"github.com/way-platform/tachograph-go/internal/card"
	"github.com/way-platform/tachograph-go/internal/vu"
	tachographv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/v1"
	"google.golang.org/protobuf/proto"
)

// Authenticate performs cryptographic authentication on a raw file with default options.
//
// This is a convenience function that uses default options:
// - Mutate: false (creates a copy before authenticating)
// - CertificateResolver: nil (uses default embedded certificate resolver)
//
// For custom options, use AuthenticateOptions directly:
//
//	opts := AuthenticateOptions{Mutate: true}
//	authenticatedRaw, err := opts.Authenticate(ctx, rawFile)
func Authenticate(ctx context.Context, rawFile *tachographv1.RawFile) (*tachographv1.RawFile, error) {
	opts := AuthenticateOptions{
		Mutate: false,
	}
	return opts.Authenticate(ctx, rawFile)
}

// AuthenticateOptions configures the signature authentication process.
type AuthenticateOptions struct {
	// CertificateResolver is used to resolve CA certificates by their Certificate Authority Reference (CAR).
	// If nil, this defaults to using DefaultCertificateResolver.
	CertificateResolver CertificateResolver

	// VerificationTime overrides the timestamp used for VU certificate validity
	// checks. The zero value uses the signed CurrentDateTime from the VU Overview
	// transfer. Callers may supply an independently recorded download time when
	// available. Card certificate validity checks do not currently use this field.
	VerificationTime time.Time

	// Mutate controls whether authentication modifies the input RawFile in-place.
	//
	// If false (default), the input RawFile is deep cloned before authentication,
	// ensuring the original remains unchanged. This is the safe default for most use cases.
	//
	// If true, the input RawFile is modified in-place, which is more efficient
	// but requires the caller to be aware that the input will be mutated.
	Mutate bool
}

// Authenticate performs cryptographic authentication on a raw tachograph file,
// populating Authentication fields in the raw records.
//
// By default (Mutate: false), this method returns a new authenticated RawFile,
// leaving the input unchanged. Set Mutate: true for in-place authentication.
//
// The zero value of AuthenticateOptions uses the default certificate resolver
// and does not mutate the input.
func (o AuthenticateOptions) Authenticate(ctx context.Context, rawFile *tachographv1.RawFile) (*tachographv1.RawFile, error) {
	if rawFile == nil {
		return nil, fmt.Errorf("rawFile cannot be nil")
	}

	// Clone the input unless mutate is explicitly requested
	var target *tachographv1.RawFile
	if o.Mutate {
		target = rawFile
	} else {
		target = proto.Clone(rawFile).(*tachographv1.RawFile)
	}

	// Convert top-level options to internal options
	cardOpts := card.AuthenticateOptions{
		CertificateResolver: o.CertificateResolver,
	}
	vuOpts := vu.AuthenticateOptions{
		CertificateResolver: o.CertificateResolver,
		VerificationTime:    o.VerificationTime,
	}

	switch target.GetType() {
	case tachographv1.RawFile_CARD:
		if err := cardOpts.AuthenticateRawCardFile(ctx, target.GetCard()); err != nil {
			return nil, err
		}
	case tachographv1.RawFile_VEHICLE_UNIT:
		if err := vuOpts.AuthenticateRawVehicleUnitFile(ctx, target.GetVehicleUnit()); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported file type: %v", target.GetType())
	}

	return target, nil
}
