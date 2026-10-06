package gorp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// HTTPError represents a non-2xx HTTP response error.
type HTTPError struct {
	StatusCode int
	Response   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("status code error: %d\n%s", e.StatusCode, e.Response)
}

// HTTPInterceptor is a callback for capturing HTTP request/response details for logging or debugging.
type HTTPInterceptor func(req *HTTPRequest, resp *HTTPResponse)

// HTTPRequest contains HTTP request details passed to an HTTPInterceptor.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// HTTPResponse contains HTTP response details passed to an HTTPInterceptor.
type HTTPResponse struct {
	StatusCode int
	Status     string
	Headers    map[string]string
	Body       string
	Duration   time.Duration
}

// Client is a ReportPortal REST API client.
type Client struct {
	project     string
	http        *resty.Client
	interceptor HTTPInterceptor
}

// NewClient creates a new Client.
//
//   - host: ReportPortal server URL (e.g. "https://reportportal.example.com")
//   - project: project name
//   - apiKey: user API key from the ReportPortal profile page
func NewClient(host, project, apiKey string) *Client {
	client := &Client{project: project}

	// Normalise host: strip trailing slashes and a stray "/api" suffix so
	// callers passing "https://rp.example.com/" or "https://rp.example.com/api"
	// all produce the same base URL.
	baseHost := strings.TrimRight(host, "/")
	baseHost = strings.TrimSuffix(baseHost, "/api")

	http := resty.New().
		SetBaseURL(baseHost + "/api").
		SetTimeout(30 * time.Second).
		SetAuthToken(apiKey).
		SetAuthScheme("bearer").
		SetDebug(false).
		OnAfterResponse(func(c *resty.Client, rs *resty.Response) error {
			if client.interceptor != nil {
				req := rs.Request

				reqHeaders := make(map[string]string)
				for k, v := range req.Header {
					if len(v) > 0 {
						reqHeaders[k] = v[0]
					}
				}

				respHeaders := make(map[string]string)
				for k, v := range rs.Header() {
					if len(v) > 0 {
						respHeaders[k] = v[0]
					}
				}

				reqBody := ""
				if req.Body != nil {
					switch b := req.Body.(type) {
					case []byte:
						reqBody = string(b)
					case string:
						reqBody = b
					default:
						if jsonBytes, err := json.Marshal(req.Body); err == nil {
							reqBody = string(jsonBytes)
						}
					}
				}

				client.interceptor(
					&HTTPRequest{
						Method:  req.Method,
						URL:     req.URL,
						Headers: reqHeaders,
						Body:    reqBody,
					},
					&HTTPResponse{
						StatusCode: rs.StatusCode(),
						Status:     rs.Status(),
						Headers:    respHeaders,
						Body:       string(rs.Body()),
						Duration:   rs.Time(),
					},
				)
			}

			if (rs.StatusCode() / 100) >= 4 { //nolint:mnd
				return &HTTPError{StatusCode: rs.StatusCode(), Response: rs.String()}
			}
			return nil
		})

	client.http = http
	return client
}

// SetDebug enables or disables debug logging via the HTTP interceptor.
// We intentionally do NOT forward to resty's own debug mode because resty
// prints the Authorization header in plaintext. Our interceptor already
// masks the header via maskAuth.
func (c *Client) SetDebug(debug bool) {
	// resty debug is kept off unconditionally; our interceptor handles it.
	_ = debug
}

// GetDebug returns the current debug mode.
func (c *Client) GetDebug() bool { return c.http.Debug }

// SetHTTPInterceptor sets a callback to intercept every HTTP request/response.
func (c *Client) SetHTTPInterceptor(interceptor HTTPInterceptor) { c.interceptor = interceptor }

// StartLaunch starts a new launch in ReportPortal.
func (c *Client) StartLaunch(launch *StartLaunchRQ) (*EntryCreatedRS, error) {
	return c.startLaunch(launch)
}

// StartLaunchRaw starts a new launch using a raw JSON body.
func (c *Client) StartLaunchRaw(body json.RawMessage) (*EntryCreatedRS, error) {
	return c.startLaunch(body)
}

func (c *Client) startLaunch(body interface{}) (*EntryCreatedRS, error) {
	var rs EntryCreatedRS
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetBody(body).
		SetResult(&rs).
		Post("v1/{project}/launch")
	return &rs, err
}

// FinishLaunch finishes a launch in ReportPortal.
func (c *Client) FinishLaunch(id string, launch *FinishExecutionRQ) (*FinishLaunchRS, error) {
	return c.finishLaunch(id, launch)
}

// FinishLaunchRaw finishes a launch using a raw JSON body.
func (c *Client) FinishLaunchRaw(id string, body json.RawMessage) (*FinishLaunchRS, error) {
	return c.finishLaunch(id, body)
}

func (c *Client) finishLaunch(id string, body interface{}) (*FinishLaunchRS, error) {
	var rs FinishLaunchRS
	_, err := c.http.R().
		SetPathParams(map[string]string{"project": c.project, "launchId": id}).
		SetBody(body).
		SetResult(&rs).
		Put("/v2/{project}/launch/{launchId}/finish")
	return &rs, err
}

// StopLaunch forces a launch to finish immediately.
func (c *Client) StopLaunch(id string) (*MsgRS, error) {
	var rs MsgRS
	_, err := c.http.R().
		SetPathParams(map[string]string{"project": c.project, "launchId": id}).
		SetBody(&FinishExecutionRQ{EndTime: NewTimestamp(time.Now()), Status: Statuses.Stopped}).
		SetResult(&rs).
		Put("/v2/{project}/launch/{launchId}/stop")
	return &rs, err
}

// StartTest starts a new top-level test item.
func (c *Client) StartTest(item *StartTestRQ) (*EntryCreatedRS, error) {
	return c.startTest(item)
}

// StartTestRaw starts a new top-level test item using a raw JSON body.
func (c *Client) StartTestRaw(body json.RawMessage) (*EntryCreatedRS, error) {
	return c.startTest(body)
}

func (c *Client) startTest(body interface{}) (*EntryCreatedRS, error) {
	var rs EntryCreatedRS
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetBody(body).
		SetResult(&rs).
		Post("/v2/{project}/item")
	return &rs, err
}

// StartChildTest starts a child test item under the given parent item ID.
func (c *Client) StartChildTest(parent string, item *StartTestRQ) (*EntryCreatedRS, error) {
	return c.startChildTest(parent, item)
}

// StartChildTestRaw starts a child test item using a raw JSON body.
func (c *Client) StartChildTestRaw(parent string, body json.RawMessage) (*EntryCreatedRS, error) {
	return c.startChildTest(parent, body)
}

func (c *Client) startChildTest(parent string, body interface{}) (*EntryCreatedRS, error) {
	var rs EntryCreatedRS
	_, err := c.http.R().
		SetPathParams(map[string]string{"project": c.project, "itemId": parent}).
		SetBody(body).
		SetResult(&rs).
		Post("/v2/{project}/item/{itemId}")
	return &rs, err
}

// FinishTest finishes a test item.
func (c *Client) FinishTest(id string, rq *FinishTestRQ) (*MsgRS, error) {
	return c.finishTest(id, rq)
}

// FinishTestRaw finishes a test item using a raw JSON body.
func (c *Client) FinishTestRaw(id string, body json.RawMessage) (*MsgRS, error) {
	return c.finishTest(id, body)
}

func (c *Client) finishTest(id string, body interface{}) (*MsgRS, error) {
	var rs MsgRS
	_, err := c.http.R().
		SetPathParams(map[string]string{"project": c.project, "itemId": id}).
		SetBody(body).
		SetResult(&rs).
		Put("/v2/{project}/item/{itemId}")
	return &rs, err
}

// SaveLog attaches a single log entry to a test item.
func (c *Client) SaveLog(log *SaveLogRQ) (*EntryCreatedRS, error) {
	var rs EntryCreatedRS
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetBody(log).
		SetResult(&rs).
		Post("/v2/{project}/log")
	return &rs, err
}

// SaveLogs saves multiple log entries in a single batch request.
func (c *Client) SaveLogs(logs ...*SaveLogRQ) (*EntryCreatedRS, error) {
	return c.SaveLogMultipart(logs, nil)
}

// SaveLogMultipart saves a batch of log entries with optional file attachments.
//
// Example:
//
//	f, _ := os.Open("screenshot.png")
//	logs := []*SaveLogRQ{{
//	    ItemID:     itemID,
//	    LaunchUUID: launchID,
//	    Level:      gorp.LogLevelError,
//	    LogTime:    gorp.NewTimestamp(time.Now()),
//	    Message:    "Test failed",
//	    Attachment: gorp.Attachment{Name: "screenshot.png"},
//	}}
//	files := []gorp.Multipart{
//	    &gorp.ReaderMultipart{FileName: "screenshot.png", ContentType: "image/png", Reader: f},
//	}
//	client.SaveLogMultipart(logs, files)
func (c *Client) SaveLogMultipart(logs []*SaveLogRQ, files []Multipart) (*EntryCreatedRS, error) {
	var bodyBuf bytes.Buffer
	if err := json.NewEncoder(&bodyBuf).Encode(logs); err != nil {
		return nil, fmt.Errorf("unable to encode log payload: %w", err)
	}

	rq := c.http.R().SetPathParam("project", c.project)
	rq.SetMultipartField("json_request_part", "", "application/json", &bodyBuf)

	for _, v := range files {
		fileName, contentType, reader, err := v.Load()
		if err != nil {
			return nil, fmt.Errorf("unable to read multipart: %w", err)
		}
		if fileName == "" {
			return nil, errMultipartFilename
		}
		rq.SetMultipartField("file", fileName, contentType, reader)
	}

	var rs EntryCreatedRS
	if _, err := rq.SetResult(&rs).Post("/v2/{project}/log"); err != nil {
		return nil, fmt.Errorf("unable to send log: %w", err)
	}
	return &rs, nil
}

// GetLaunchesByFilter retrieves launches matching the given filter params.
func (c *Client) GetLaunchesByFilter(filter map[string]string) (*LaunchPage, error) {
	var launches LaunchPage
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetResult(&launches).
		SetQueryParams(filter).
		Get("/v2/{project}/launch")
	return &launches, err
}

// GetLaunchesByFilterString retrieves launches using a raw RP filter query string.
func (c *Client) GetLaunchesByFilterString(filter string) (*LaunchPage, error) {
	var launches LaunchPage
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetResult(&launches).
		SetQueryString(filter).
		Get("/v2/{project}/launch")
	return &launches, err
}

// GetLaunchesByFilterName retrieves launches matching a saved filter by name.
func (c *Client) GetLaunchesByFilterName(name string) (*LaunchPage, error) {
	filter, err := c.GetFiltersByName(name)
	if err != nil {
		return nil, err
	}
	if filter.Page.Size < 1 || len(filter.Content) == 0 {
		return nil, fmt.Errorf("no filter %q found", name) //nolint:err113
	}
	var launches LaunchPage
	_, err = c.http.R().
		SetPathParam("project", c.project).
		SetResult(&launches).
		SetQueryParams(ConvertToFilterParams(filter.Content[0])).
		Get("/v2/{project}/launch")
	return &launches, err
}

// GetFiltersByName retrieves a saved filter by name.
func (c *Client) GetFiltersByName(name string) (*FilterPage, error) {
	var filter FilterPage
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetQueryParam("filter.eq.name", name).
		SetResult(&filter).
		Get("/v2/{project}/filter")
	return &filter, err
}

// MergeLaunches merges two or more launches.
func (c *Client) MergeLaunches(rq *MergeLaunchesRQ) (*LaunchResource, error) {
	var rs LaunchResource
	_, err := c.http.R().
		SetPathParam("project", c.project).
		SetBody(rq).
		SetResult(&rs).
		Post("/v2/{project}/launch/merge")
	return &rs, err
}
