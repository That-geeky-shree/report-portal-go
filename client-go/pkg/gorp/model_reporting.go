package gorp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Log level constants for use with SaveLogRQ.
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
	LogLevelFatal = "FATAL"
)

const defaultDateTimeFormat = "2006-01-02T15:04:05.999-0700"

// knownDateTimeFormats are tried in order when the primary format fails.
var knownDateTimeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	defaultDateTimeFormat,
}

// StartLaunchRQ is the request payload for starting a launch.
type StartLaunchRQ struct {
	StartRQ
	Mode    LaunchMode `json:"mode"`
	Rerun   bool       `json:"rerun,omitempty"`
	RerunOf *uuid.UUID `json:"rerunOf,omitempty"`
}

// FinishTestRQ is the request payload for finishing a test item.
type FinishTestRQ struct {
	FinishExecutionRQ
	LaunchUUID string `json:"launchUuid,omitempty"`
	TestCaseID string `json:"testCaseId,omitempty"`
	Retry      bool   `json:"retry,omitempty"`
	RetryOf    string `json:"retryOf,omitempty"`
}

// SaveLogRQ is the request payload for saving a log entry.
type SaveLogRQ struct {
	LaunchUUID string     `json:"launchUuid,omitempty"`
	ItemID     string     `json:"itemUuid,omitempty"`
	LogTime    Timestamp  `json:"time,omitempty"`
	Message    string     `json:"message,omitempty"`
	Level      string     `json:"level,omitempty"`
	Attachment Attachment `json:"file,omitempty"`
}

// StartTestRQ is the request payload for starting a test item.
type StartTestRQ struct {
	StartRQ
	CodeRef         string       `json:"codeRef,omitempty"`
	Parameters      []*Parameter `json:"parameters,omitempty"`
	UniqueID        string       `json:"uniqueId,omitempty"`
	TestCaseID      string       `json:"testCaseId,omitempty"`
	TestReferenceId string       `json:"testReferenceId,omitempty"`
	LaunchID        string       `json:"launchUuid,omitempty"`
	Type            TestItemType `json:"type,omitempty"`
	Retry           bool         `json:"retry,omitempty"`
	HasStats        *bool        `json:"hasStats,omitempty"`
	Priority        *int         `json:"priority,omitempty"`
}

// FinishExecutionRQ is the request payload for finishing a launch or test item.
type FinishExecutionRQ struct {
	EndTime            Timestamp           `json:"endTime,omitempty"`
	Status             Status              `json:"status,omitempty"`
	Description        string              `json:"description,omitempty"`
	Attributes         []*Attribute        `json:"attributes,omitempty"`
	SdkExecutionStatus *SdkExecutionStatus `json:"sdkExecutionStatus,omitempty"`
}

// EntryCreatedRS is the response payload for any create operation.
type EntryCreatedRS struct {
	ID string `json:"id,omitempty"`
}

// FinishLaunchRS is the response payload for finishing a launch.
type FinishLaunchRS struct {
	EntryCreatedRS
	Number int64 `json:"number,omitempty"`
}

// MsgRS is a generic success response payload.
type MsgRS struct {
	Msg string `json:"msg,omitempty"`
}

// StartRQ is the common base for start requests.
type StartRQ struct {
	UUID        *uuid.UUID   `json:"uuid,omitempty"`
	Name        string       `json:"name,omitempty"`
	Description string       `json:"description,omitempty"`
	Attributes  []*Attribute `json:"attributes,omitempty"`
	StartTime   Timestamp    `json:"startTime,omitempty"`
}

// Attribute represents a ReportPortal key-value attribute.
// Use Key+Value for tag-style attributes, or Key+Value+System for system attributes.
type Attribute struct {
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
	System bool   `json:"system,omitempty"`
}

// Parameter represents a key-value test parameter.
type Parameter struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Attachment represents a file attached to a log entry.
type Attachment struct {
	Name string `json:"name,omitempty"`
}

// Timestamp wraps time.Time to marshal/unmarshal as epoch milliseconds.
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes epoch milliseconds, RFC3339, or the default datetime format.
// A JSON null value results in the zero Timestamp (not an error).
func (rt *Timestamp) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		rt.Time = time.Time{}
		return nil
	}
	trimmed := strings.Trim(s, "\"")

	// Try epoch milliseconds first.
	if msInt, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		rt.Time = time.Unix(0, msInt*int64(time.Millisecond))
		return nil
	}
	// Try known string formats (RFC3339Nano handles Z and +00:00).
	for _, layout := range knownDateTimeFormats {
		if dt, err := time.Parse(layout, trimmed); err == nil {
			rt.Time = dt
			return nil
		}
	}
	return fmt.Errorf("cannot parse timestamp %q", trimmed)
}

// MarshalJSON encodes the timestamp as epoch milliseconds.
// A zero time marshals as null so that optional end-time fields are omitted correctly.
func (rt Timestamp) MarshalJSON() ([]byte, error) {
	if rt.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(rt.Time.In(time.UTC).UnixNano()/int64(time.Millisecond), 10)), nil
}

// NewTimestamp wraps a time.Time as a Timestamp.
func NewTimestamp(t time.Time) Timestamp { return Timestamp{Time: t} }

// Stats holds pass/fail/skip counts for a test group.
type Stats struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// PriorityStatus holds Stats broken down by P0–P4 priority.
type PriorityStatus struct {
	P0 Stats `json:"0"`
	P1 Stats `json:"1"`
	P2 Stats `json:"2"`
	P3 Stats `json:"3"`
	P4 Stats `json:"4"`
}

// GetStats returns the Stats pointer for the given priority level (0–4), or nil.
func (ps *PriorityStatus) GetStats(priority int) *Stats {
	switch priority {
	case 0:
		return &ps.P0
	case 1:
		return &ps.P1
	case 2:
		return &ps.P2
	case 3:
		return &ps.P3
	case 4:
		return &ps.P4
	default:
		return nil
	}
}

// SdkExecutionStatus holds the aggregated execution statistics sent when finishing a launch.
type SdkExecutionStatus struct {
	PriorityStatus PriorityStatus `json:"priorityStatus"`
	OverallStatus  Stats          `json:"overallStatus"`
}

// NewSdkExecutionStatus returns an empty SdkExecutionStatus.
func NewSdkExecutionStatus() *SdkExecutionStatus {
	return &SdkExecutionStatus{}
}
