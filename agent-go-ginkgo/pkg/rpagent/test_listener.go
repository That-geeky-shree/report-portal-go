package rpagent

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reportportal/client-go/pkg/gorp"
)

// isRetriableError reports whether an error warrants a retry.
// Retries on: network errors, HTTP 5xx, HTTP 429.
// Does NOT retry on: 4xx client errors (except 429).
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	var httpErr *gorp.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}
	s := err.Error()
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "EOF") ||
		strings.Contains(s, "no such host") ||
		strings.Contains(s, "network is unreachable")
}

// retry runs f up to attempts times with exponential backoff + jitter.
// It stops early on non-retriable errors.
func retry(attempts int, baseDelay time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}
		if !isRetriableError(err) {
			return err
		}
		if i == attempts-1 {
			break
		}
		backoff := baseDelay * time.Duration(1<<uint(i))
		jitter := time.Duration(rand.Int63n(int64(backoff)/2 + 1)) //nolint:gosec
		log.Printf("retrying in %v (attempt %d/%d): %v", backoff+jitter, i+2, attempts, err)
		time.Sleep(backoff + jitter)
	}
	return fmt.Errorf("after %d attempts, last error: %w", attempts, err)
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// TestListener wraps the ReportPortal client with retry logic and convenience methods.
type TestListener struct {
	client     *gorp.Client
	launchUuid string
	launchName string
}

// NewTestListener creates a TestListener.
func NewTestListener(client *gorp.Client, launchUuid, launchName string) *TestListener {
	return &TestListener{client: client, launchUuid: launchUuid, launchName: launchName}
}

// FinishSuite finishes a suite item.
func (tl *TestListener) FinishSuite(name string, status gorp.Status, id string) {
	if id == "" {
		log.Printf("no suite ID for %q, skipping finish", name)
		return
	}
	_, err := tl.client.FinishTest(id, &gorp.FinishTestRQ{
		FinishExecutionRQ: gorp.FinishExecutionRQ{EndTime: gorp.NewTimestamp(time.Now()), Status: status},
		LaunchUUID:        tl.launchUuid,
	})
	if err != nil {
		log.Printf("failed to finish suite %q: %v", name, err)
	}
}

// StartTest starts a test item (no retry-ID variant).
func (tl *TestListener) StartTest(name, parentUUID string, priority *int, parameters []*gorp.Parameter, itemType gorp.TestItemType) string {
	return tl.startItem(name, parentUUID, priority, parameters, itemType, "")
}

// StartTestWithReferenceId starts a test item with an external reference ID.
func (tl *TestListener) StartTestWithReferenceId(name, parentUUID string, priority *int, parameters []*gorp.Parameter, itemType gorp.TestItemType, refID string) string {
	return tl.startItem(name, parentUUID, priority, parameters, itemType, refID)
}

// StartTestWithStats starts a test item with an explicit hasStats flag.
// Pass false for By() child steps so RP does not count them in launch totals.
func (tl *TestListener) StartTestWithStats(name, parentUUID string, priority *int, parameters []*gorp.Parameter, itemType gorp.TestItemType, hasStats bool) string {
	return tl.startItemWithStats(name, parentUUID, priority, parameters, itemType, "", hasStats)
}

func (tl *TestListener) startItem(name, parentUUID string, priority *int, parameters []*gorp.Parameter, itemType gorp.TestItemType, refID string) string {
	return tl.startItemWithStats(name, parentUUID, priority, parameters, itemType, refID, true)
}

func (tl *TestListener) startItemWithStats(name, parentUUID string, priority *int, parameters []*gorp.Parameter, itemType gorp.TestItemType, refID string, hasStats bool) string {
	// Generate a stable client UUID for this start request so that retries are
	// idempotent: if the server created the item but the response was lost, a
	// retry with the same UUID is deduplicated by RP instead of creating a
	// duplicate item.
	clientUUID := uuid.New()

	// Build a stable identity key from the full item path (parentUUID + name)
	// so that distinct items with the same leaf name don't share a testCaseId.
	testCaseKey := name
	if parentUUID != "" {
		testCaseKey = parentUUID + "/" + name
	}

	rq := &gorp.StartTestRQ{
		StartRQ: gorp.StartRQ{
			UUID:      &clientUUID,
			Name:      name,
			StartTime: gorp.NewTimestamp(time.Now()),
		},
		LaunchID:        tl.launchUuid,
		HasStats:        boolPtr(hasStats),
		UniqueID:        clientUUID.String(),
		CodeRef:         testCaseKey,
		TestCaseID:      testCaseKey,
		Type:            itemType,
		Parameters:      parameters,
		TestReferenceId: refID,
	}

	if priority != nil {
		rq.Priority = intPtr(*priority)
		// RP UI reads priority from attributes; the standalone field is not displayed.
		rq.Attributes = append(rq.Attributes, &gorp.Attribute{
			Key:   "priority",
			Value: fmt.Sprintf("P%d", *priority),
		})
	}

	if refID != "" {
		// RP UI reads testReferenceId from attributes; the standalone field is not displayed.
		rq.Attributes = append(rq.Attributes, &gorp.Attribute{
			Key:   "testReferenceId",
			Value: refID,
		})
	}

	if itemType == gorp.TestItemTypes.Suite {
		rq.CodeRef = ""
		if refID == "" {
			rq.TestCaseID = ""
		}
	}

	var rs *gorp.EntryCreatedRS
	err := retry(3, 200*time.Millisecond, func() error {
		var e error
		if parentUUID != "" {
			rs, e = tl.client.StartChildTest(parentUUID, rq)
		} else {
			rs, e = tl.client.StartTest(rq)
		}
		return e
	})
	if err != nil {
		log.Printf("WARNING: failed to start test %q after retries: %v", name, err)
		return ""
	}
	return rs.ID
}

// FinishTestWithPriority finishes a test item.
func (tl *TestListener) FinishTestWithPriority(name string, status gorp.Status, id string, _ *int) {
	if id == "" {
		log.Printf("no test ID for %q, skipping finish", name)
		return
	}
	err := retry(3, 200*time.Millisecond, func() error {
		_, e := tl.client.FinishTest(id, &gorp.FinishTestRQ{
			FinishExecutionRQ: gorp.FinishExecutionRQ{EndTime: gorp.NewTimestamp(time.Now()), Status: status},
			LaunchUUID:        tl.launchUuid,
		})
		return e
	})
	if err != nil {
		log.Printf("WARNING: failed to finish test %q after retries: %v", name, err)
	}
}

// SendLog sends a log entry to the given test item.
func (tl *TestListener) SendLog(itemID, level, message string) error {
	return retry(3, 200*time.Millisecond, func() error {
		_, e := tl.client.SaveLog(&gorp.SaveLogRQ{
			ItemID:     itemID,
			LaunchUUID: tl.launchUuid,
			Level:      level,
			LogTime:    gorp.NewTimestamp(time.Now()),
			Message:    message,
		})
		return e
	})
}

// SendAttachmentLog streams a file to ReportPortal as a multipart log attachment.
func (tl *TestListener) SendAttachmentLog(itemID, level, message, filePath, originalFilename string) error {
	file, err := os.Open(filePath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to open attachment %s: %v — original message: %s", originalFilename, err, message)
		log.Printf("ERROR: %s", errMsg)
		return tl.SendLog(itemID, gorp.LogLevelError, errMsg)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return tl.SendLog(itemID, gorp.LogLevelError, fmt.Sprintf("failed to stat attachment %s: %v", originalFilename, err))
	}

	// Detect content type from first 512 bytes without loading the whole file.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := file.Seek(0, 0); err != nil {
		return tl.SendLog(itemID, gorp.LogLevelError, fmt.Sprintf("failed to seek attachment %s: %v", originalFilename, err))
	}

	LogVerboseOperation(fmt.Sprintf("sending attachment: %s (%s, %d bytes)", originalFilename, contentType, fileInfo.Size()))

	logRQ := &gorp.SaveLogRQ{
		ItemID:     itemID,
		LaunchUUID: tl.launchUuid,
		Level:      level,
		LogTime:    gorp.NewTimestamp(time.Now()),
		Message:    message,
		Attachment: gorp.Attachment{Name: originalFilename},
	}
	multipart := &gorp.ReaderMultipart{
		FileName:    originalFilename,
		ContentType: contentType,
		Reader:      file,
	}

	err = retry(3, 200*time.Millisecond, func() error {
		if _, se := file.Seek(0, 0); se != nil {
			return se
		}
		_, e := tl.client.SaveLogMultipart([]*gorp.SaveLogRQ{logRQ}, []gorp.Multipart{multipart})
		return e
	})
	if err != nil {
		log.Printf("ERROR: multipart upload failed after retries: %v", err)
		return tl.SendLog(itemID, level, fmt.Sprintf("%s\n[attachment %s could not be uploaded]", message, originalFilename))
	}

	os.Remove(filePath) // clean up temp copy
	return nil
}

// SendLaunchLog sends a log entry at the launch level (not tied to a test item).
func (tl *TestListener) SendLaunchLog(level, message string) error {
	_, err := tl.client.SaveLog(&gorp.SaveLogRQ{
		LaunchUUID: tl.launchUuid,
		Level:      level,
		LogTime:    gorp.NewTimestamp(time.Now()),
		Message:    message,
	})
	return err
}

// MapGinkgoStateToStatus converts a Ginkgo spec state string to a ReportPortal Status.
// All five Ginkgo failure states map to Failed or Interrupted:
//
//	failed, panicked → Failed
//	interrupted, aborted, timedout → Interrupted (test was running, not a logic failure)
func MapGinkgoStateToStatus(state string) gorp.Status {
	switch strings.ToLower(state) {
	case "passed":
		return gorp.Statuses.Passed
	case "failed", "panicked":
		return gorp.Statuses.Failed
	case "interrupted", "aborted", "timedout":
		return gorp.Statuses.Interrupted
	case "skipped", "pending":
		return gorp.Statuses.Skipped
	default:
		return gorp.Statuses.Skipped
	}
}

// sendLaunchLog sends a log entry at the launch level (package-level convenience, used by wrapper).
func sendLaunchLog(client *gorp.Client, launchID, level, message string) {
	client.SaveLog(&gorp.SaveLogRQ{ //nolint:errcheck
		LaunchUUID: launchID,
		Level:      level,
		LogTime:    gorp.NewTimestamp(time.Now()),
		Message:    message,
	})
}
