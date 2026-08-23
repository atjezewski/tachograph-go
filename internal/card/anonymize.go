package card

import (
	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	"google.golang.org/protobuf/proto"
)

// AnonymizeOptions configures the anonymization of card files.
type AnonymizeOptions struct {
	// PreserveDistanceAndTrips controls whether distance and trip data are preserved.
	PreserveDistanceAndTrips bool

	// PreserveTimestamps controls whether timestamps are preserved.
	PreserveTimestamps bool
}

// AnonymizeDriverCardFile creates an anonymized copy of a driver card file.
func (opts AnonymizeOptions) AnonymizeDriverCardFile(file *cardv1.DriverCardFile) (*cardv1.DriverCardFile, error) {
	if file == nil {
		return nil, nil
	}

	// Clone the file to avoid mutating the input
	result := proto.Clone(file).(*cardv1.DriverCardFile)

	// Anonymize common EFs (Master File)
	if icc := result.GetIcc(); icc != nil {
		result.SetIcc(opts.anonymizeIcc(icc))
	}
	if ic := result.GetIc(); ic != nil {
		result.SetIc(opts.anonymizeIc(ic))
	}

	// Anonymize Gen1 DF (Tachograph)
	if tachograph := result.GetTachograph(); tachograph != nil {
		// Clear certificates (TLV format: set to nil to omit blocks entirely)
		tachograph.SetCardCertificate(nil)
		tachograph.SetCaCertificate(nil)

		if appId := tachograph.GetApplicationIdentification(); appId != nil {
			tachograph.SetApplicationIdentification(opts.anonymizeApplicationIdentification(appId))
		}
		if identification := tachograph.GetIdentification(); identification != nil {
			tachograph.SetIdentification(opts.anonymizeDriverCardIdentification(identification))
		}
		if drivingLicenceInfo := tachograph.GetDrivingLicenceInfo(); drivingLicenceInfo != nil {
			tachograph.SetDrivingLicenceInfo(opts.anonymizeDrivingLicenceInfo(drivingLicenceInfo))
		}
		if eventsData := tachograph.GetEventsData(); eventsData != nil {
			tachograph.SetEventsData(opts.anonymizeEventsData(eventsData))
		}
		if faultsData := tachograph.GetFaultsData(); faultsData != nil {
			tachograph.SetFaultsData(opts.anonymizeFaultsData(faultsData))
		}
		if driverActivityData := tachograph.GetDriverActivityData(); driverActivityData != nil {
			tachograph.SetDriverActivityData(opts.anonymizeDriverActivityData(driverActivityData))
		}
		if vehiclesUsed := tachograph.GetVehiclesUsed(); vehiclesUsed != nil {
			tachograph.SetVehiclesUsed(opts.anonymizeVehiclesUsed(vehiclesUsed))
		}
		if places := tachograph.GetPlaces(); places != nil {
			tachograph.SetPlaces(opts.anonymizePlaces(places))
		}
		if currentUsage := tachograph.GetCurrentUsage(); currentUsage != nil {
			tachograph.SetCurrentUsage(opts.anonymizeCurrentUsage(currentUsage))
		}
		if controlActivityData := tachograph.GetControlActivityData(); controlActivityData != nil {
			tachograph.SetControlActivityData(opts.anonymizeControlActivityData(controlActivityData))
		}
		if specificConditions := tachograph.GetSpecificConditions(); specificConditions != nil {
			tachograph.SetSpecificConditions(opts.anonymizeSpecificConditions(specificConditions))
		}
	}

	// Anonymize Gen2 DF (Tachograph_G2)
	if tachographG2 := result.GetTachographG2(); tachographG2 != nil {
		// Clear certificates (TLV format: set to nil to omit blocks entirely)
		tachographG2.SetCardMaCertificate(nil)
		tachographG2.SetCardSignCertificate(nil)
		tachographG2.SetCaCertificate(nil)
		tachographG2.SetLinkCertificate(nil)

		// Anonymize Gen2 versions of shared EFs
		if appIdG2 := tachographG2.GetApplicationIdentification(); appIdG2 != nil {
			tachographG2.SetApplicationIdentification(opts.anonymizeApplicationIdentificationG2(appIdG2))
		}
		if identification := tachographG2.GetIdentification(); identification != nil {
			tachographG2.SetIdentification(opts.anonymizeDriverCardIdentification(identification))
		}
		if drivingLicenceInfo := tachographG2.GetDrivingLicenceInfo(); drivingLicenceInfo != nil {
			tachographG2.SetDrivingLicenceInfo(opts.anonymizeDrivingLicenceInfo(drivingLicenceInfo))
		}
		if eventsData := tachographG2.GetEventsData(); eventsData != nil {
			tachographG2.SetEventsData(opts.anonymizeEventsData(eventsData))
		}
		if faultsData := tachographG2.GetFaultsData(); faultsData != nil {
			tachographG2.SetFaultsData(opts.anonymizeFaultsData(faultsData))
		}
		if driverActivityData := tachographG2.GetDriverActivityData(); driverActivityData != nil {
			tachographG2.SetDriverActivityData(opts.anonymizeDriverActivityData(driverActivityData))
		}
		if vehiclesUsedG2 := tachographG2.GetVehiclesUsed(); vehiclesUsedG2 != nil {
			tachographG2.SetVehiclesUsed(opts.anonymizeVehiclesUsedG2(vehiclesUsedG2))
		}
		if placesG2 := tachographG2.GetPlaces(); placesG2 != nil {
			tachographG2.SetPlaces(opts.anonymizePlacesG2(placesG2))
		}
		if currentUsage := tachographG2.GetCurrentUsage(); currentUsage != nil {
			tachographG2.SetCurrentUsage(opts.anonymizeCurrentUsage(currentUsage))
		}
		if controlActivityData := tachographG2.GetControlActivityData(); controlActivityData != nil {
			tachographG2.SetControlActivityData(opts.anonymizeControlActivityData(controlActivityData))
		}
		if specificConditionsG2 := tachographG2.GetSpecificConditions(); specificConditionsG2 != nil {
			tachographG2.SetSpecificConditions(opts.anonymizeSpecificConditionsG2(specificConditionsG2))
		}

		// Anonymize Gen2-exclusive EFs
		if vehicleUnitsUsed := tachographG2.GetVehicleUnitsUsed(); vehicleUnitsUsed != nil {
			tachographG2.SetVehicleUnitsUsed(opts.anonymizeVehicleUnitsUsed(vehicleUnitsUsed))
		}
		if gnssPlaces := tachographG2.GetGnssPlaces(); gnssPlaces != nil {
			tachographG2.SetGnssPlaces(opts.anonymizeGnssPlaces(gnssPlaces))
		}

		// Anonymize Gen2v2-exclusive EFs
		if placesAuth := tachographG2.GetPlacesAuthentication(); placesAuth != nil {
			tachographG2.SetPlacesAuthentication(opts.anonymizePlacesAuthentication(placesAuth))
		}
		if gnssPlacesAuth := tachographG2.GetGnssPlacesAuthentication(); gnssPlacesAuth != nil {
			tachographG2.SetGnssPlacesAuthentication(opts.anonymizeGnssPlacesAuthentication(gnssPlacesAuth))
		}
		if borderCrossings := tachographG2.GetBorderCrossings(); borderCrossings != nil {
			tachographG2.SetBorderCrossings(opts.anonymizeBorderCrossings(borderCrossings))
		}
		if loadUnload := tachographG2.GetLoadUnloadOperations(); loadUnload != nil {
			tachographG2.SetLoadUnloadOperations(opts.anonymizeLoadUnloadOperations(loadUnload))
		}
		if loadTypeEntries := tachographG2.GetLoadTypeEntries(); loadTypeEntries != nil {
			tachographG2.SetLoadTypeEntries(opts.anonymizeLoadTypeEntries(loadTypeEntries))
		}
	}

	return result, nil
}
