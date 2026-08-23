package card

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	cardv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/card/v1"
	ddv1 "github.com/way-platform/tachograph-go/proto/gen/go/wayplatform/connect/tachograph/dd/v1"
)

func TestActivity_Generation1(t *testing.T) {
	// Discover all matching hexdump files using type-safe enums
	hexdumpFiles, err := findHexdumpFiles(
		cardv1.ElementaryFileType_EF_DRIVER_ACTIVITY_DATA,
		ddv1.Generation_GENERATION_1,
		cardv1.ContentType_DATA,
	)
	if err != nil {
		t.Fatalf("Failed to discover hexdump files: %v", err)
	}
	if len(hexdumpFiles) == 0 {
		t.Fatal("No hexdump files found for EF_DRIVER_ACTIVITY_DATA GENERATION_1")
	}

	// Run subtest for each discovered file
	for _, hexdumpPath := range hexdumpFiles {
		// Use relative path from testdata as subtest name
		relPath := strings.TrimPrefix(hexdumpPath, "testdata/records/")
		testName := strings.TrimSuffix(relPath, ".hexdump")

		t.Run(testName, func(t *testing.T) {
			// Read hexdump
			data, err := readHexdump(hexdumpPath)
			if err != nil {
				t.Fatalf("Failed to read hexdump: %v", err)
			}

			// Unmarshal
			opts := UnmarshalOptions{}
			activity, err := opts.unmarshalDriverActivityData(data)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Golden JSON comparison
			goldenPath := goldenJSONPath(hexdumpPath)
			loadOrCreateGolden(t, activity, goldenPath)

			// Round-trip test
			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalDriverActivity(activity)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if diff := cmp.Diff(data, marshaled); diff != "" {
				t.Errorf("Binary round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestActivity_Generation2(t *testing.T) {
	// Discover all matching hexdump files using type-safe enums
	hexdumpFiles, err := findHexdumpFiles(
		cardv1.ElementaryFileType_EF_DRIVER_ACTIVITY_DATA,
		ddv1.Generation_GENERATION_2,
		cardv1.ContentType_DATA,
	)
	if err != nil {
		t.Fatalf("Failed to discover hexdump files: %v", err)
	}
	if len(hexdumpFiles) == 0 {
		t.Fatal("No hexdump files found for EF_DRIVER_ACTIVITY_DATA GENERATION_2")
	}

	// Run subtest for each discovered file
	for _, hexdumpPath := range hexdumpFiles {
		// Use relative path from testdata as subtest name
		relPath := strings.TrimPrefix(hexdumpPath, "testdata/records/")
		testName := strings.TrimSuffix(relPath, ".hexdump")

		t.Run(testName, func(t *testing.T) {
			// Read hexdump
			data, err := readHexdump(hexdumpPath)
			if err != nil {
				t.Fatalf("Failed to read hexdump: %v", err)
			}

			// Unmarshal
			opts := UnmarshalOptions{}
			activity, err := opts.unmarshalDriverActivityData(data)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Golden JSON comparison
			goldenPath := goldenJSONPath(hexdumpPath)
			loadOrCreateGolden(t, activity, goldenPath)

			// Round-trip test
			marshalOpts := MarshalOptions{}
			marshaled, err := marshalOpts.MarshalDriverActivity(activity)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if diff := cmp.Diff(data, marshaled); diff != "" {
				t.Errorf("Binary round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestActivity_RecordHeaderWrapsBufferEnd covers a full cyclic buffer whose
// oldest daily record starts two bytes before the end, so that its four-byte
// header wraps around to the front of the buffer.
//
// Reading that header without wrapping ends the backwards walk early and
// silently drops every record older than it. On a card that has been in use
// long enough to fill its buffer this is most of the driver's history.
func TestActivity_RecordHeaderWrapsBufferEnd(t *testing.T) {
	const (
		ringSize  = 58
		oldestPos = 56 // header spans bytes 56, 57, 0, 1
		newestPos = 40

		drivingAt0800 = 0x19E0 // driver slot, single, card inserted, driving, 480 minutes
		workAt1000    = 0x1258 // driver slot, single, card inserted, work, 600 minutes
	)

	day := func(d int) time.Time {
		return time.Date(2025, time.January, d, 0, 0, 0, 0, time.UTC)
	}

	// Four consecutive days laid out so the ring is exactly full, chained
	// 56 -> 12 -> 26 -> 40 and back around to 56.
	ring := make([]byte, ringSize)
	writeWrappedActivityRecord(ring, oldestPos, activityDailyRecordBytes(16, day(1), 0x0001, 10, drivingAt0800))
	writeWrappedActivityRecord(ring, 12, activityDailyRecordBytes(14, day(2), 0x0002, 20, drivingAt0800))
	writeWrappedActivityRecord(ring, 26, activityDailyRecordBytes(14, day(3), 0x0003, 30, drivingAt0800))
	writeWrappedActivityRecord(ring, newestPos, activityDailyRecordBytes(14, day(4), 0x0004, 40, drivingAt0800, workAt1000))

	data := binary.BigEndian.AppendUint16(nil, oldestPos)
	data = binary.BigEndian.AppendUint16(data, newestPos)
	data = append(data, ring...)

	opts := UnmarshalOptions{}
	activity, err := opts.unmarshalDriverActivityData(data)
	if err != nil {
		t.Fatalf("unmarshal driver activity data: %v", err)
	}

	records := activity.GetDailyRecords()
	if len(records) != 4 {
		t.Fatalf("got %d daily records, want 4: the record whose header wraps must not end the walk", len(records))
	}

	wantDates := []string{"2025-01-01", "2025-01-02", "2025-01-03", "2025-01-04"}
	wantPresence := []int32{1, 2, 3, 4}
	wantDistance := []int32{10, 20, 30, 40}
	wantChanges := []int{1, 1, 1, 2}
	for i, record := range records {
		if !record.GetValid() {
			t.Errorf("record %d is not valid", i)
			continue
		}
		if got := record.GetActivityRecordDate().AsTime().UTC().Format(time.DateOnly); got != wantDates[i] {
			t.Errorf("record %d date = %s, want %s", i, got, wantDates[i])
		}
		if got := record.GetActivityDailyPresenceCounter().GetValue(); got != wantPresence[i] {
			t.Errorf("record %d presence counter = %d, want %d", i, got, wantPresence[i])
		}
		if got := record.GetActivityDayDistance(); got != wantDistance[i] {
			t.Errorf("record %d day distance = %d, want %d", i, got, wantDistance[i])
		}
		if got := len(record.GetActivityChangeInfo()); got != wantChanges[i] {
			t.Errorf("record %d has %d activity changes, want %d", i, got, wantChanges[i])
		}
	}

	// The record that wraps must decode like any other, not just be counted.
	if changes := records[0].GetActivityChangeInfo(); len(changes) == 1 {
		if got := changes[0].GetActivity(); got != ddv1.DriverActivityValue_DRIVING {
			t.Errorf("wrapped record activity = %v, want DRIVING", got)
		}
		if got := changes[0].GetTimeOfChangeMinutes(); got != 480 {
			t.Errorf("wrapped record time of change = %d, want 480", got)
		}
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalDriverActivity(activity)
	if err != nil {
		t.Fatalf("marshal driver activity: %v", err)
	}
	if diff := cmp.Diff(data, marshaled); diff != "" {
		t.Errorf("binary round-trip mismatch (-want +got):\n%s", diff)
	}
}

// activityDailyRecordBytes builds one CardActivityDailyRecord: the length of the
// physically preceding record, this record's own length, then its payload.
func activityDailyRecordBytes(previousRecordLength int, date time.Time, presenceCounter uint16, distanceKm uint16, changes ...uint16) []byte {
	record := make([]byte, 12+2*len(changes))
	binary.BigEndian.PutUint16(record[0:2], uint16(previousRecordLength))
	binary.BigEndian.PutUint16(record[2:4], uint16(len(record)))
	binary.BigEndian.PutUint32(record[4:8], uint32(date.Unix()))
	binary.BigEndian.PutUint16(record[8:10], presenceCounter) // BCD
	binary.BigEndian.PutUint16(record[10:12], distanceKm)
	for i, change := range changes {
		binary.BigEndian.PutUint16(record[12+2*i:14+2*i], change)
	}
	return record
}

// writeWrappedActivityRecord places a record in the cyclic buffer at pos,
// wrapping around the end of the buffer.
func writeWrappedActivityRecord(buffer []byte, pos int, record []byte) {
	for i, b := range record {
		buffer[(pos+i)%len(buffer)] = b
	}
}

// TestActivity_ZeroValuedChangeIsARecord covers a daily record whose first
// activity change is '0000'H.
//
// That value decodes to driver slot, single crew, card inserted, break/rest at
// 00:00 — the ordinary way a day opens, and every daily set must carry the
// status at midnight. Treating it as an empty slot drops a real activity
// transition and leaves the parsed record disagreeing with its own
// activityRecordLength.
func TestActivity_ZeroValuedChangeIsARecord(t *testing.T) {
	const (
		breakRestAtMidnight = 0x0000
		drivingAt0800       = 0x19E0
	)

	record := activityDailyRecordBytes(0, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		0x0001, 42, breakRestAtMidnight, drivingAt0800)
	data := binary.BigEndian.AppendUint16(nil, 0) // oldest record pointer
	data = binary.BigEndian.AppendUint16(data, 0) // newest record pointer
	data = append(data, record...)

	opts := UnmarshalOptions{}
	activity, err := opts.unmarshalDriverActivityData(data)
	if err != nil {
		t.Fatalf("unmarshal driver activity data: %v", err)
	}
	records := activity.GetDailyRecords()
	if len(records) != 1 {
		t.Fatalf("got %d daily records, want 1", len(records))
	}

	changes := records[0].GetActivityChangeInfo()
	if want := int(records[0].GetActivityRecordLength()-12) / 2; len(changes) != want {
		t.Fatalf("got %d activity changes, want %d: the record length accounts for every change", len(changes), want)
	}
	midnight := changes[0]
	if got := midnight.GetSlot(); got != ddv1.CardSlotNumber_DRIVER_SLOT {
		t.Errorf("slot = %v, want DRIVER_SLOT", got)
	}
	if midnight.GetCrew() {
		t.Error("crew = true, want false")
	}
	if !midnight.GetInserted() {
		t.Error("inserted = false, want true")
	}
	if got := midnight.GetActivity(); got != ddv1.DriverActivityValue_BREAK_REST {
		t.Errorf("activity = %v, want BREAK_REST", got)
	}
	if got := midnight.GetTimeOfChangeMinutes(); got != 0 {
		t.Errorf("time of change = %d, want 0", got)
	}

	marshalOpts := MarshalOptions{}
	marshaled, err := marshalOpts.MarshalDriverActivity(activity)
	if err != nil {
		t.Fatalf("marshal driver activity: %v", err)
	}
	if diff := cmp.Diff(data, marshaled); diff != "" {
		t.Errorf("binary round-trip mismatch (-want +got):\n%s", diff)
	}
}
