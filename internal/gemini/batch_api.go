package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BatchJobState values from Gemini Batch API.
const (
	BatchStatePending   = "JOB_STATE_PENDING"
	BatchStateRunning   = "JOB_STATE_RUNNING"
	BatchStateSucceeded = "JOB_STATE_SUCCEEDED"
	BatchStateFailed    = "JOB_STATE_FAILED"
	BatchStateCancelled = "JOB_STATE_CANCELLED"
	BatchStateExpired   = "JOB_STATE_EXPIRED"
)

// BatchJob tracks an async Gemini batch prediction job.
type BatchJob struct {
	Name            string
	State           string
	OutputFile      string
	Error           string
	InlineResponses []BatchInlineResponse
}

// BatchInlineResponse is one inline batch output row.
type BatchInlineResponse struct {
	Key      string
	Response []byte
	Error    string
}

// BatchJSONLLine is one row in a Gemini Batch API input JSONL file.
type BatchJSONLLine struct {
	Key     string          `json:"key"`
	Request json.RawMessage `json:"request"`
}

type batchJSONLLine = BatchJSONLLine

const batchInlineMaxBytes = 18 << 20 // Gemini inline batch limit is 20MB; stay under.

type createBatchRequest struct {
	Batch createBatchBody `json:"batch"`
}

type createBatchBody struct {
	DisplayName string           `json:"displayName"`
	InputConfig batchInputConfig `json:"inputConfig"`
}

type batchInputConfig struct {
	FileName string              `json:"fileName,omitempty"`
	Requests *batchInlineRequest `json:"requests,omitempty"`
}

type batchInlineRequest struct {
	Requests []batchInlineItem `json:"requests"`
}

type batchInlineItem struct {
	Request  json.RawMessage   `json:"request"`
	Metadata map[string]string `json:"metadata"`
}

type uploadFileResponse struct {
	File struct {
		Name string `json:"name"`
		URI  string `json:"uri"`
	} `json:"file"`
}

type batchJobResponse struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Dest  struct {
		FileName         string `json:"fileName"`
		InlinedResponses []struct {
			Metadata struct {
				Key string `json:"key"`
			} `json:"metadata"`
			Response *generateResponse `json:"response"`
			Error    *apiError         `json:"error"`
		} `json:"inlinedResponses"`
	} `json:"dest"`
	Output struct {
		ResponsesFile    string `json:"responsesFile"`
		InlinedResponses []struct {
			Metadata struct {
				Key string `json:"key"`
			} `json:"metadata"`
			Response *generateResponse `json:"response"`
			Error    *apiError         `json:"error"`
		} `json:"inlinedResponses"`
	} `json:"output"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
	Error *apiError `json:"error"`
}

func (c *Client) batchModelPath() string {
	model := strings.TrimPrefix(strings.TrimSpace(c.model), "models/")
	return "models/" + model
}

// UploadBatchJSONL uploads a JSONL input file for batch processing.
func (c *Client) UploadBatchJSONL(ctx context.Context, data []byte, displayName string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("gemini client nil")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("gemini batch: empty upload")
	}
	startURL := fmt.Sprintf("%s/upload/v1beta/files?key=%s", strings.TrimSuffix(c.baseURL, "/v1beta"), c.apiKey)
	meta := fmt.Sprintf(`{"file":{"displayName":"%s"}}`, escapeJSONString(displayName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, strings.NewReader(meta))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("X-Goog-Upload-Header-Content-Type", "application/jsonl")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	uploadURL := resp.Header.Get("X-Goog-Upload-URL")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if uploadURL == "" {
		return "", fmt.Errorf("gemini batch upload start failed: %s", prettyHTTPBody(resp.StatusCode, body))
	}

	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	upReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	upReq.Header.Set("X-Goog-Upload-Offset", "0")
	upReq.Header.Set("X-Goog-Upload-Command", "upload, finalize")

	upResp, err := c.httpClient.Do(upReq)
	if err != nil {
		return "", err
	}
	upBody, err := io.ReadAll(upResp.Body)
	upResp.Body.Close()
	if err != nil {
		return "", err
	}
	if upResp.StatusCode < 200 || upResp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini batch upload finalize failed: %s", prettyHTTPBody(upResp.StatusCode, upBody))
	}
	var parsed uploadFileResponse
	if err := json.Unmarshal(upBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini batch upload decode: %w", err)
	}
	if parsed.File.Name == "" {
		return "", fmt.Errorf("gemini batch upload: missing file name")
	}
	return parsed.File.Name, nil
}

// CreateBatchFromFile submits a batch job using an uploaded JSONL file.
func (c *Client) CreateBatchFromFile(ctx context.Context, fileName, displayName string) (*BatchJob, error) {
	if c == nil {
		return nil, fmt.Errorf("gemini client nil")
	}
	body, err := json.Marshal(createBatchRequest{
		Batch: createBatchBody{
			DisplayName: displayName,
			InputConfig: batchInputConfig{FileName: fileName},
		},
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s:batchGenerateContent?key=%s", c.baseURL, c.batchModelPath(), c.apiKey)
	respBody, err := c.postJSON(ctx, url, body)
	if err != nil {
		return nil, err
	}
	return parseBatchJobResponse(respBody)
}

// SubmitBatchJSONL uploads or inlines a JSONL payload depending on size.
func (c *Client) SubmitBatchJSONL(ctx context.Context, lines []BatchJSONLLine, displayName string) (*BatchJob, error) {
	if c == nil {
		return nil, fmt.Errorf("gemini client nil")
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("gemini batch: no requests")
	}
	var buf bytes.Buffer
	parsed := make([]BatchJSONLLine, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line.Key) == "" || len(line.Request) == 0 {
			continue
		}
		row, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}
		buf.Write(row)
		buf.WriteByte('\n')
		parsed = append(parsed, line)
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("gemini batch: no valid lines")
	}
	if buf.Len() <= batchInlineMaxBytes {
		return c.CreateBatchInline(ctx, parsed, displayName)
	}
	fileName, err := c.UploadBatchJSONL(ctx, buf.Bytes(), displayName)
	if err != nil {
		return nil, err
	}
	return c.CreateBatchFromFile(ctx, fileName, displayName)
}

// CreateBatchInline submits a batch job with inline requests (under 20MB total).
func (c *Client) CreateBatchInline(ctx context.Context, lines []BatchJSONLLine, displayName string) (*BatchJob, error) {
	if c == nil {
		return nil, fmt.Errorf("gemini client nil")
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("gemini batch: no inline requests")
	}
	items := make([]batchInlineItem, 0, len(lines))
	for _, line := range lines {
		key := strings.TrimSpace(line.Key)
		if key == "" || len(line.Request) == 0 {
			continue
		}
		items = append(items, batchInlineItem{
			Request:  line.Request,
			Metadata: map[string]string{"key": key},
		})
	}
	body, err := json.Marshal(createBatchRequest{
		Batch: createBatchBody{
			DisplayName: displayName,
			InputConfig: batchInputConfig{
				Requests: &batchInlineRequest{Requests: items},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s:batchGenerateContent?key=%s", c.baseURL, c.batchModelPath(), c.apiKey)
	respBody, err := c.postJSON(ctx, url, body)
	if err != nil {
		return nil, err
	}
	return parseBatchJobResponse(respBody)
}

// GetBatchJob polls batch job status.
func (c *Client) GetBatchJob(ctx context.Context, name string) (*BatchJob, error) {
	if c == nil {
		return nil, fmt.Errorf("gemini client nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("gemini batch: empty job name")
	}
	url := fmt.Sprintf("%s/%s?key=%s", c.baseURL, name, c.apiKey)
	respBody, err := c.getJSON(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseBatchJobResponse(respBody)
}

// DownloadBatchFile downloads a Gemini File API object (batch output JSONL).
func (c *Client) DownloadBatchFile(ctx context.Context, fileName string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("gemini client nil")
	}
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, fmt.Errorf("gemini batch: empty file name")
	}
	base := strings.TrimSuffix(c.baseURL, "/v1beta")
	url := fmt.Sprintf("%s/download/v1beta/%s:download?alt=media&key=%s", base, fileName, c.apiKey)
	return c.getJSON(ctx, url)
}

func (c *Client) postJSON(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, parseAPIError(resp.StatusCode, respBody, retryAfter)
	}
	return respBody, nil
}

func (c *Client) getJSON(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, parseAPIError(resp.StatusCode, respBody, retryAfter)
	}
	return respBody, nil
}

func parseBatchJobResponse(body []byte) (*BatchJob, error) {
	var raw batchJobResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("gemini batch decode: %w", err)
	}
	job := &BatchJob{Name: raw.Name}
	job.State = firstNonEmpty(raw.State, raw.Metadata.State)
	job.OutputFile = firstNonEmpty(raw.Dest.FileName, raw.Output.ResponsesFile)
	if raw.Error != nil {
		job.Error = raw.Error.Message
	}
	inline := raw.Dest.InlinedResponses
	if len(inline) == 0 {
		inline = raw.Output.InlinedResponses
	}
	for _, row := range inline {
		item := BatchInlineResponse{Key: row.Metadata.Key}
		if row.Error != nil {
			item.Error = row.Error.Message
		} else if row.Response != nil {
			item.Response = batchResponseText(row.Response)
		}
		job.InlineResponses = append(job.InlineResponses, item)
	}
	return job, nil
}

func batchResponseText(resp *generateResponse) []byte {
	if resp == nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil
	}
	return []byte(strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func prettyHTTPBody(code int, body []byte) string {
	return fmt.Sprintf("http %d: %s", code, truncateBytes(body, 300))
}

func truncateBytes(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	if len(b) < 2 {
		return s
	}
	return string(b[1 : len(b)-1])
}
