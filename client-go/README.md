# client-go

A Go HTTP client for the [ReportPortal](https://reportportal.io) REST API.

Used internally by [agent-go-ginkgo](https://github.com/reportportal/agent-go-ginkgo) but usable as a standalone library if you want to build your own ReportPortal integration in Go.

## Requirements

- Go 1.21+
- A running ReportPortal instance (v5+)

## Installation

```bash
go get github.com/reportportal/client-go
```

## Quick Start

```go
import (
    "time"
    "github.com/reportportal/client-go/pkg/gorp"
)

client := gorp.NewClient(
    "https://your-reportportal-host",
    "your_project_name",
    "your_api_key",
)

// Start a launch
launchRS, err := client.StartLaunch(&gorp.StartLaunchRQ{
    StartRQ: gorp.StartRQ{
        Name:      "My Launch",
        StartTime: gorp.NewTimestamp(time.Now()),
    },
    Mode: gorp.LaunchModes.Default,
})

// Start a suite item
suiteRS, err := client.StartTest(&gorp.StartTestRQ{
    StartRQ:  gorp.StartRQ{Name: "My Suite", StartTime: gorp.NewTimestamp(time.Now())},
    LaunchID: launchRS.ID,
    Type:     gorp.TestItemTypes.Suite,
    HasStats: boolPtr(true),
})

// Start a test item inside the suite
testRS, err := client.StartChildTest(suiteRS.ID, &gorp.StartTestRQ{
    StartRQ:  gorp.StartRQ{Name: "should do something", StartTime: gorp.NewTimestamp(time.Now())},
    LaunchID: launchRS.ID,
    Type:     gorp.TestItemTypes.Step,
    HasStats: boolPtr(true),
})

// Send a log entry
client.SaveLog(&gorp.SaveLogRQ{
    ItemID:     testRS.ID,
    LaunchUUID: launchRS.ID,
    Level:      gorp.LogLevelInfo,
    LogTime:    gorp.NewTimestamp(time.Now()),
    Message:    "test passed",
})

// Finish the test item
client.FinishTest(testRS.ID, &gorp.FinishTestRQ{
    FinishExecutionRQ: gorp.FinishExecutionRQ{
        EndTime: gorp.NewTimestamp(time.Now()),
        Status:  gorp.Statuses.Passed,
    },
    LaunchUUID: launchRS.ID,
})

// Finish the launch
client.FinishLaunch(launchRS.ID, &gorp.FinishExecutionRQ{
    EndTime: gorp.NewTimestamp(time.Now()),
    Status:  gorp.Statuses.Passed,
})
```

## API Reference

### Client

```go
// NewClient creates a client. The endpoint is normalised: trailing slashes and
// a trailing "/api" suffix are stripped automatically, so both
// "https://rp.example.com" and "https://rp.example.com/api/" work.
// A 30-second HTTP timeout is set by default.
client := gorp.NewClient(endpoint, project, apiKey string) *Client

// SetHTTPInterceptor installs a resty request middleware (used for custom
// auth headers, debug logging, etc.).
client.SetHTTPInterceptor(fn resty.RequestMiddleware)
```

### Launches

```go
client.StartLaunch(rq *StartLaunchRQ) (*EntryCreatedRS, error)
client.FinishLaunch(id string, rq *FinishExecutionRQ) (*MsgRS, error)
client.StopLaunch(id string) (*MsgRS, error)
client.GetLaunchesByFilter(params map[string]string) (*LaunchPage, error)
client.GetLaunchesByFilterName(name string) (*LaunchPage, error)
client.MergeLaunches(rq *MergeLaunchesRQ) (*LaunchResource, error)
```

### Test Items

```go
client.StartTest(rq *StartTestRQ) (*EntryCreatedRS, error)
client.StartChildTest(parentID string, rq *StartTestRQ) (*EntryCreatedRS, error)
client.FinishTest(id string, rq *FinishTestRQ) (*MsgRS, error)
```

### Logs

```go
client.SaveLog(rq *SaveLogRQ) (*EntryCreatedRS, error)
client.SaveLogs(rqs []*SaveLogRQ) (*EntryCreatedRS, error)
client.SaveLogMultipart(logs []*SaveLogRQ, attachments []Multipart) (*EntryCreatedRS, error)
```

### Timestamp

`gorp.Timestamp` wraps `time.Time` with custom JSON marshalling:

- **Marshal:** formats as RFC3339 milliseconds (`2006-01-02T15:04:05.000Z`). A zero `time.Time` marshals as JSON `null`.
- **Unmarshal:** accepts RFC3339 with nanoseconds (Z suffix), standard RFC3339, and the RP milliseconds format. JSON `null` unmarshals as a zero `time.Time`.

```go
gorp.NewTimestamp(time.Now())   // wrap time.Time
gorp.NewTimestamp(time.Time{})  // marshals as null
```

### Pagination (sort)

`LaunchQueryParams` supports multiple sort fields. Pass them as a slice — they are joined as repeated `page.sort=` query parameters as required by the RP API:

```go
params := gorp.LaunchQueryParams{
    Sort: []string{"startTime,DESC", "name,ASC"},
}
```

### Constants

```go
// Statuses
gorp.Statuses.Passed
gorp.Statuses.Failed
gorp.Statuses.Skipped
gorp.Statuses.Interrupted

// Test item types
gorp.TestItemTypes.Suite
gorp.TestItemTypes.Test
gorp.TestItemTypes.Step

// Launch modes
gorp.LaunchModes.Default
gorp.LaunchModes.Debug

// Log levels
gorp.LogLevelDebug
gorp.LogLevelInfo
gorp.LogLevelWarn
gorp.LogLevelError
gorp.LogLevelFatal
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
