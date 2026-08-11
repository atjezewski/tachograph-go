package card

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/way-platform/tachograph-go/internal/cert"
	"github.com/way-platform/tachograph-go/internal/security"
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
	securityv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/security/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthenticateOptions configures the card authentication process.
type AuthenticateOptions struct {
	// CertificateResolver is used to resolve CA certificates by their Certificate Authority Reference (CAR).
	CertificateResolver cert.Resolver

	// VerificationTime is an independently recorded card acquisition time used
	// for certificate validity checks. The zero value skips temporal checks
	// because EF_Card_Download is unsigned.
	VerificationTime time.Time
}

// AuthenticateRawCardFile performs cryptographic authentication on all signed Elementary Files
// in a raw card file. For each signed EF, it verifies the signature in the following signature
// block and populates the authentication field.
//
// This function mutates the rawFile parameter by setting the authentication field
// on signed EF records.
//
// Authentication is performed by:
// 1. Extracting the card certificate from the appropriate EF (e.g., EF_Card_Certificate)
// 2. Verifying the certificate chain against the European Root CA
// 3. Verifying the EF data signature using the card certificate
//
// If any EF fails authentication, its authentication.status will be set to the
// appropriate error status, and this function will return an error after processing
// all EFs.
func (opts AuthenticateOptions) AuthenticateRawCardFile(ctx context.Context, rawFile *cardv1.RawCardFile) error {
	if rawFile == nil {
		return fmt.Errorf("rawFile cannot be nil")
	}
	if opts.CertificateResolver == nil {
		opts.CertificateResolver = cert.DefaultResolver()
	}

	// Check which generations are present in the file
	// Many cards have both Gen1 and Gen2 data for backward compatibility
	hasGen1, hasGen2 := opts.detectGenerations(rawFile)
	cardType := InferFileType(rawFile)
	if cardType == cardv1.CardType_CARD_TYPE_UNSPECIFIED {
		return fmt.Errorf("unable to determine card type")
	}

	var errs []error

	// Authenticate Gen1 data if present
	if hasGen1 {
		if err := opts.authenticateGen1CardFile(ctx, rawFile, cardType); err != nil {
			errs = append(errs, fmt.Errorf("Gen1 authentication failed: %w", err))
		}
	}

	// Authenticate Gen2 data if present
	if hasGen2 {
		if err := opts.authenticateGen2CardFile(ctx, rawFile, cardType); err != nil {
			errs = append(errs, fmt.Errorf("Gen2 authentication failed: %w", err))
		}
	}

	if !hasGen1 && !hasGen2 {
		return fmt.Errorf("unable to determine card generation")
	}

	return errors.Join(errs...)
}

// authenticateGen1CardFile authenticates a Generation 1 card file using RSA signature recovery.
func (opts AuthenticateOptions) authenticateGen1CardFile(ctx context.Context, rawFile *cardv1.RawCardFile, cardType cardv1.CardType) error {
	// Step 1: Extract certificates
	cardCert, mscaCert, err := opts.extractGen1CardCertificates(rawFile)
	if err != nil {
		markCardCertificateFailure(rawFile, ddv1.Generation_GENERATION_1)
		return fmt.Errorf("failed to extract Gen1 certificates: %w", err)
	}

	// Step 2: Verify certificate chain
	rootCert, err := opts.verifyGen1CertificateChain(ctx, cardCert, mscaCert, cardType)
	if err != nil {
		markCardCertificateFailure(rawFile, ddv1.Generation_GENERATION_1)
		return fmt.Errorf("certificate chain verification failed: %w", err)
	}

	// Step 3: Authenticate each signed EF
	var errs []error
	records := rawFile.GetRecords()

	for i := 0; i < len(records); i++ {
		record := records[i]

		// Skip if not Gen1 or not a data record
		if record.GetGeneration() != ddv1.Generation_GENERATION_1 {
			continue
		}
		if record.GetContentType() != cardv1.ContentType_DATA {
			continue
		}

		// Skip unsigned EFs
		if !isSignedEF(record.GetFile()) {
			continue
		}

		// Look for signature record (next record with same FID but signature type)
		var signatureRecord *cardv1.RawCardFile_Record
		if i+1 < len(records) {
			nextRecord := records[i+1]
			if nextRecord.GetFile() == record.GetFile() &&
				nextRecord.GetContentType() == cardv1.ContentType_SIGNATURE &&
				nextRecord.GetGeneration() == record.GetGeneration() {
				signatureRecord = nextRecord
			}
		}

		// Verify signature
		if err := opts.authenticateGen1EF(record, signatureRecord, cardCert, rootCert); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// authenticateGen2CardFile authenticates a Generation 2 card file using ECDSA signature verification.
func (opts AuthenticateOptions) authenticateGen2CardFile(ctx context.Context, rawFile *cardv1.RawCardFile, cardType cardv1.CardType) error {
	// Step 1: Extract certificates
	cardCert, mscaCert, linkCert, err := opts.extractGen2CardCertificates(rawFile)
	if err != nil {
		markCardCertificateFailure(rawFile, ddv1.Generation_GENERATION_2)
		setLinkCertificateAuthentication(rawFile, securityv1.Authentication_CERTIFICATE_VERIFICATION_FAILED)
		return fmt.Errorf("failed to extract Gen2 certificates: %w", err)
	}

	// Step 2: Verify certificate chain
	rootCert, err := opts.verifyGen2CertificateChain(ctx, cardCert, mscaCert, linkCert, cardType)
	if err != nil {
		markCardCertificateFailure(rawFile, ddv1.Generation_GENERATION_2)
		setLinkCertificateAuthentication(rawFile, securityv1.Authentication_CERTIFICATE_VERIFICATION_FAILED)
		return fmt.Errorf("certificate chain verification failed: %w", err)
	}
	if linkCert != nil {
		setLinkCertificateAuthentication(rawFile, securityv1.Authentication_VERIFIED)
	}

	// Step 3: Authenticate each signed EF
	var errs []error
	records := rawFile.GetRecords()

	for i := 0; i < len(records); i++ {
		record := records[i]

		// Skip if not Gen2 or not a data record
		if record.GetGeneration() != ddv1.Generation_GENERATION_2 {
			continue
		}
		if record.GetContentType() != cardv1.ContentType_DATA {
			continue
		}

		// Skip unsigned EFs
		if !isSignedEF(record.GetFile()) {
			continue
		}

		// Look for signature record (next record with same FID but signature type)
		var signatureRecord *cardv1.RawCardFile_Record
		if i+1 < len(records) {
			nextRecord := records[i+1]
			if nextRecord.GetFile() == record.GetFile() &&
				nextRecord.GetContentType() == cardv1.ContentType_SIGNATURE &&
				nextRecord.GetGeneration() == record.GetGeneration() {
				signatureRecord = nextRecord
			}
		}

		// Verify signature
		if err := opts.authenticateGen2EF(record, signatureRecord, cardCert, rootCert); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// detectGenerations checks which generations are present in the card file.
// Returns (hasGen1, hasGen2) booleans.
// Many cards have both Gen1 and Gen2 data for backward compatibility.
func (opts AuthenticateOptions) detectGenerations(rawFile *cardv1.RawCardFile) (bool, bool) {
	hasGen1 := false
	hasGen2 := false

	for _, record := range rawFile.GetRecords() {
		switch record.GetGeneration() {
		case ddv1.Generation_GENERATION_1:
			hasGen1 = true
		case ddv1.Generation_GENERATION_2:
			hasGen2 = true
		}

		// Early exit if we found both
		if hasGen1 && hasGen2 {
			return hasGen1, hasGen2
		}
	}

	return hasGen1, hasGen2
}

// extractGen1CardCertificates extracts the card and MSCA certificates from a Gen1 card file.
func (opts AuthenticateOptions) extractGen1CardCertificates(rawFile *cardv1.RawCardFile) (*securityv1.RsaCertificate, *securityv1.RsaCertificate, error) {
	var cardCert, mscaCert *securityv1.RsaCertificate
	var err error

	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() != ddv1.Generation_GENERATION_1 {
			continue
		}

		switch record.GetFile() {
		case cardv1.ElementaryFileType_EF_CARD_CERTIFICATE:
			cardCert, err = security.UnmarshalRsaCertificate(record.GetValue())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse card certificate: %w", err)
			}

		case cardv1.ElementaryFileType_EF_CA_CERTIFICATE:
			mscaCert, err = security.UnmarshalRsaCertificate(record.GetValue())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse MSCA certificate: %w", err)
			}
		}
	}

	if cardCert == nil {
		return nil, nil, fmt.Errorf("card certificate not found in Gen1 card file")
	}
	if mscaCert == nil {
		return nil, nil, fmt.Errorf("MSCA certificate not found in Gen1 card file")
	}

	return cardCert, mscaCert, nil
}

// extractGen2CardCertificates extracts the card and MSCA certificates from a Gen2 card file.
func (opts AuthenticateOptions) extractGen2CardCertificates(rawFile *cardv1.RawCardFile) (*securityv1.EccCertificate, *securityv1.EccCertificate, *securityv1.EccCertificate, error) {
	var cardCert, mscaCert, linkCert *securityv1.EccCertificate
	var err error

	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() != ddv1.Generation_GENERATION_2 {
			continue
		}

		switch record.GetFile() {
		case cardv1.ElementaryFileType_EF_CARD_SIGN_CERTIFICATE:
			cardCert, err = security.UnmarshalEccCertificate(record.GetValue())
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse card certificate: %w", err)
			}

		case cardv1.ElementaryFileType_EF_CA_CERTIFICATE:
			mscaCert, err = security.UnmarshalEccCertificate(record.GetValue())
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse MSCA certificate: %w", err)
			}

		case cardv1.ElementaryFileType_EF_LINK_CERTIFICATE:
			if isZeroFilled(record.GetValue()) {
				continue
			}
			linkCert, err = security.UnmarshalEccCertificate(record.GetValue())
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse link certificate: %w", err)
			}
		}
	}

	if cardCert == nil {
		return nil, nil, nil, fmt.Errorf("card certificate not found in Gen2 card file")
	}
	if mscaCert == nil {
		return nil, nil, nil, fmt.Errorf("MSCA certificate not found in Gen2 card file")
	}

	return cardCert, mscaCert, linkCert, nil
}

// verifyGen1CertificateChain verifies the Gen1 certificate chain: EUR Root -> MSCA -> Card.
func (opts AuthenticateOptions) verifyGen1CertificateChain(ctx context.Context, cardCert *securityv1.RsaCertificate, mscaCert *securityv1.RsaCertificate, cardType cardv1.CardType) (*securityv1.RootCertificate, error) {
	// Get EUR root certificate
	rootCert, err := opts.CertificateResolver.GetRootCertificate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get root certificate: %w", err)
	}

	if opts.VerificationTime.IsZero() {
		err = security.VerifyRsaCertificateWithRoot(mscaCert, rootCert)
	} else {
		err = security.VerifyRsaCertificateWithRootAt(mscaCert, rootCert, opts.VerificationTime)
	}
	if err != nil {
		return nil, fmt.Errorf("MSCA certificate verification failed: %w", err)
	}
	if err := security.VerifyRsaCertificateRole(mscaCert, security.RsaCertificateRoleMemberState); err != nil {
		return nil, fmt.Errorf("invalid MSCA certificate role: %w", err)
	}

	if opts.VerificationTime.IsZero() {
		err = security.VerifyRsaCertificateWithCA(cardCert, mscaCert)
	} else {
		err = security.VerifyRsaCertificateWithCAAt(cardCert, mscaCert, opts.VerificationTime)
	}
	if err != nil {
		return nil, fmt.Errorf("card certificate verification failed: %w", err)
	}
	role, err := gen1CardCertificateRole(cardType)
	if err != nil {
		return nil, err
	}
	if err := security.VerifyRsaCertificateRole(cardCert, role); err != nil {
		return nil, fmt.Errorf("invalid card certificate role: %w", err)
	}

	return rootCert, nil
}

// verifyGen2CertificateChain verifies the Gen2 certificate chain: EUR Root -> MSCA -> Card_Sign.
func (opts AuthenticateOptions) verifyGen2CertificateChain(ctx context.Context, cardCert *securityv1.EccCertificate, mscaCert, linkCert *securityv1.EccCertificate, cardType cardv1.CardType) (*securityv1.EccCertificate, error) {
	rootCert, err := cert.ResolveEccTrustAnchor(ctx, opts.CertificateResolver, mscaCert, linkCert, opts.VerificationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Gen2 trust anchor: %w", err)
	}
	if err := security.VerifyEccCertificateRole(mscaCert, security.EccCertificateRoleMemberStateCA); err != nil {
		return nil, fmt.Errorf("invalid MSCA certificate role: %w", err)
	}
	role, err := gen2CardCertificateRole(cardType)
	if err != nil {
		return nil, err
	}
	if err := security.VerifyEccCertificateRole(cardCert, role); err != nil {
		return nil, fmt.Errorf("invalid Card_Sign certificate role: %w", err)
	}

	if opts.VerificationTime.IsZero() {
		err = security.VerifyEccCertificateWithEccRoot(mscaCert, rootCert)
	} else {
		err = security.VerifyEccCertificateWithCAAt(mscaCert, rootCert, opts.VerificationTime)
	}
	if err != nil {
		return nil, fmt.Errorf("MSCA certificate verification failed: %w", err)
	}

	if opts.VerificationTime.IsZero() {
		err = security.VerifyEccCertificateWithCA(cardCert, mscaCert)
	} else {
		err = security.VerifyEccCertificateWithCAAt(cardCert, mscaCert, opts.VerificationTime)
	}
	if err != nil {
		return nil, fmt.Errorf("card certificate verification failed: %w", err)
	}

	return rootCert, nil
}

// authenticateGen1EF authenticates a single Gen1 EF using RSA signature verification.
func (opts AuthenticateOptions) authenticateGen1EF(dataRecord *cardv1.RawCardFile_Record, signatureRecord *cardv1.RawCardFile_Record, cardCert *securityv1.RsaCertificate, rootCert *securityv1.RootCertificate) error {
	auth := &securityv1.Authentication{}
	dataRecord.SetAuthentication(auth)
	auth.SetSignerCertificate(rsaCertificateInfo(cardCert))
	rootInfo := &securityv1.CertificateInfo{}
	rootInfo.SetCertificateHolderReference(rootCert.GetKeyId())
	auth.SetRootCertificate(rootInfo)
	if !opts.VerificationTime.IsZero() {
		auth.SetSignatureCreationTime(timestamppb.New(opts.VerificationTime))
	}

	if signatureRecord == nil {
		auth.SetStatus(securityv1.Authentication_DATA_SIGNATURE_INVALID)
		return fmt.Errorf("signature record not found for EF %v", dataRecord.GetFile())
	}

	// Verify the signature using PKCS#1 v1.5
	data := dataRecord.GetValue()
	signature := signatureRecord.GetValue()

	if err := security.VerifyRsaDataSignature(data, signature, cardCert); err != nil {
		auth.SetStatus(securityv1.Authentication_DATA_SIGNATURE_INVALID)
		return fmt.Errorf("signature verification failed for EF %v: %w", dataRecord.GetFile(), err)
	}

	// Authentication succeeded
	auth.SetStatus(securityv1.Authentication_VERIFIED)
	auth.SetSignatureAlgorithm(securityv1.SignatureAlgorithm_SHA1_WITH_RSA_ENCRYPTION)

	return nil
}

// authenticateGen2EF authenticates a single Gen2 EF using ECDSA signature verification.
func (opts AuthenticateOptions) authenticateGen2EF(dataRecord *cardv1.RawCardFile_Record, signatureRecord *cardv1.RawCardFile_Record, cardCert *securityv1.EccCertificate, rootCert *securityv1.EccCertificate) error {
	auth := &securityv1.Authentication{}
	dataRecord.SetAuthentication(auth)
	auth.SetSignerCertificate(eccCertificateInfo(cardCert))
	auth.SetRootCertificate(eccCertificateInfo(rootCert))
	if !opts.VerificationTime.IsZero() {
		auth.SetSignatureCreationTime(timestamppb.New(opts.VerificationTime))
	}

	if signatureRecord == nil {
		auth.SetStatus(securityv1.Authentication_DATA_SIGNATURE_INVALID)
		return fmt.Errorf("signature record not found for EF %v", dataRecord.GetFile())
	}

	// Verify the signature using ECDSA
	data := dataRecord.GetValue()
	signature := signatureRecord.GetValue()

	if err := security.VerifyEccDataSignature(data, signature, cardCert); err != nil {
		auth.SetStatus(securityv1.Authentication_DATA_SIGNATURE_INVALID)
		return fmt.Errorf("signature verification failed for EF %v: %w", dataRecord.GetFile(), err)
	}

	// Determine signature algorithm based on curve
	// We need to extract this from the certificate's public key
	pubKey := cardCert.GetPublicKey()
	var sigAlg securityv1.SignatureAlgorithm
	if pubKey != nil {
		oid := pubKey.GetDomainParametersOid()
		switch oid {
		case "1.3.36.3.3.2.8.1.1.7", "1.2.840.10045.3.1.7": // brainpoolP256r1, NIST P-256
			sigAlg = securityv1.SignatureAlgorithm_ECDSA_WITH_SHA256
		case "1.3.36.3.3.2.8.1.1.11", "1.3.132.0.34": // brainpoolP384r1, NIST P-384
			sigAlg = securityv1.SignatureAlgorithm_ECDSA_WITH_SHA384
		case "1.3.36.3.3.2.8.1.1.13", "1.3.132.0.35": // brainpoolP512r1, NIST P-521
			sigAlg = securityv1.SignatureAlgorithm_ECDSA_WITH_SHA512
		default:
			sigAlg = securityv1.SignatureAlgorithm_SIGNATURE_ALGORITHM_UNSPECIFIED
		}
	}

	// Authentication succeeded
	auth.SetStatus(securityv1.Authentication_VERIFIED)
	auth.SetSignatureAlgorithm(sigAlg)

	return nil
}

func gen1CardCertificateRole(cardType cardv1.CardType) (security.RsaCertificateRole, error) {
	switch cardType {
	case cardv1.CardType_DRIVER_CARD:
		return security.RsaCertificateRoleDriverCard, nil
	case cardv1.CardType_WORKSHOP_CARD:
		return security.RsaCertificateRoleWorkshopCard, nil
	case cardv1.CardType_CONTROL_CARD:
		return security.RsaCertificateRoleControlCard, nil
	case cardv1.CardType_COMPANY_CARD:
		return security.RsaCertificateRoleCompanyCard, nil
	default:
		return 0, fmt.Errorf("unsupported card type %v for Gen1 certificate role", cardType)
	}
}

func gen2CardCertificateRole(cardType cardv1.CardType) (security.EccCertificateRole, error) {
	switch cardType {
	case cardv1.CardType_DRIVER_CARD:
		return security.EccCertificateRoleDriverCardSign, nil
	case cardv1.CardType_WORKSHOP_CARD:
		return security.EccCertificateRoleWorkshopCardSign, nil
	default:
		return 0, fmt.Errorf("card type %v has no Gen2 Card_Sign certificate", cardType)
	}
}

func markCardCertificateFailure(rawFile *cardv1.RawCardFile, generation ddv1.Generation) {
	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() != generation ||
			record.GetContentType() != cardv1.ContentType_DATA ||
			!isSignedEF(record.GetFile()) {
			continue
		}
		auth := &securityv1.Authentication{}
		auth.SetStatus(securityv1.Authentication_CERTIFICATE_VERIFICATION_FAILED)
		record.SetAuthentication(auth)
	}
}

func setLinkCertificateAuthentication(rawFile *cardv1.RawCardFile, status securityv1.Authentication_Status) {
	for _, record := range rawFile.GetRecords() {
		if record.GetGeneration() != ddv1.Generation_GENERATION_2 ||
			record.GetContentType() != cardv1.ContentType_DATA ||
			record.GetFile() != cardv1.ElementaryFileType_EF_LINK_CERTIFICATE ||
			isZeroFilled(record.GetValue()) {
			continue
		}
		auth := &securityv1.Authentication{}
		auth.SetStatus(status)
		record.SetAuthentication(auth)
	}
}

func isZeroFilled(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

func rsaCertificateInfo(cert *securityv1.RsaCertificate) *securityv1.CertificateInfo {
	info := &securityv1.CertificateInfo{}
	info.SetCertificationAuthorityReference(cert.GetCertificateAuthorityReference())
	info.SetCertificateHolderReference(cert.GetCertificateHolderReference())
	info.SetValidTo(cert.GetEndOfValidity())
	return info
}

func eccCertificateInfo(cert *securityv1.EccCertificate) *securityv1.CertificateInfo {
	info := &securityv1.CertificateInfo{}
	info.SetCertificationAuthorityReference(cert.GetCertificateAuthorityReference())
	info.SetCertificateHolderReference(cert.GetCertificateHolderReference())
	info.SetValidFrom(cert.GetCertificateEffectiveDate())
	info.SetValidTo(cert.GetCertificateExpirationDate())
	info.SetCertificationAuthorityReferenceRaw(cert.GetCertificateAuthorityReferenceRaw())
	info.SetCertificateHolderReferenceRaw(cert.GetCertificateHolderReferenceRaw())
	return info
}

// isSignedEF returns true if the given EF type should have a signature.
func isSignedEF(fileType cardv1.ElementaryFileType) bool {
	// Per regulation, these EFs are NOT signed:
	// - EF_ICC, EF_IC (common card files)
	// - All certificate EFs
	// - EF_Card_Download (all variants)
	switch fileType {
	case cardv1.ElementaryFileType_EF_ICC,
		cardv1.ElementaryFileType_EF_IC,
		cardv1.ElementaryFileType_EF_CARD_CERTIFICATE,
		cardv1.ElementaryFileType_EF_CA_CERTIFICATE,
		cardv1.ElementaryFileType_EF_CARD_MA_CERTIFICATE,
		cardv1.ElementaryFileType_EF_CARD_SIGN_CERTIFICATE,
		cardv1.ElementaryFileType_EF_LINK_CERTIFICATE,
		cardv1.ElementaryFileType_EF_CARD_DOWNLOAD_DRIVER,
		cardv1.ElementaryFileType_EF_CARD_DOWNLOAD_WORKSHOP:
		return false
	default:
		return true
	}
}
