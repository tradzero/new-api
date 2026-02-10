package zhipu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	channelconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Zhipu video API structures
// ============================

type zhipuVideoRequest struct {
	Model            string `json:"model"`
	Prompt           string `json:"prompt,omitempty"`
	ImageURL         any    `json:"image_url,omitempty"`
	Quality          string `json:"quality,omitempty"`
	WithAudio        *bool  `json:"with_audio,omitempty"`
	WatermarkEnabled *bool  `json:"watermark_enabled,omitempty"`
	Size             string `json:"size,omitempty"`
	FPS              int    `json:"fps,omitempty"`
	Duration         int    `json:"duration,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
}

type zhipuVideoSubmitResponse struct {
	Model      string `json:"model"`
	ID         string `json:"id"`
	RequestID  string `json:"request_id"`
	TaskStatus string `json:"task_status"`
}

type zhipuVideoResultItem struct {
	URL           string `json:"url"`
	CoverImageURL string `json:"cover_image_url"`
}

type zhipuVideoFetchResponse struct {
	ID          string                 `json:"id"`
	RequestID   string                 `json:"request_id"`
	Created     int64                  `json:"created"`
	Model       string                 `json:"model"`
	TaskStatus  string                 `json:"task_status"`
	VideoResult []zhipuVideoResultItem `json:"video_result"`
}

// ============================
// Model pricing configuration
// ============================

// PricingFunc calculates OtherRatios for a given model based on its request parameters.
// Each model (or model family) can have its own pricing logic.
type PricingFunc func(req *relaycommon.TaskSubmitReq) map[string]float64

// pricingRegistry maps model name prefixes to their pricing functions.
// Add new entries here to support pricing for additional models.
var pricingRegistry = map[string]PricingFunc{
	// CogVideoX family: per-second billing
	"cogvideox": pricingPerSecond,

	// Sora family: per-second + resolution multiplier
	"sora-2-pro": pricingSoraV2,
	"sora-2":     pricingSoraV2,
}

// pricingPerSecond bills by duration only (default for cogvideox models).
func pricingPerSecond(req *relaycommon.TaskSubmitReq) map[string]float64 {
	duration := 5
	if req.Duration > 0 {
		duration = req.Duration
	}
	return map[string]float64{
		"seconds": float64(duration),
	}
}

// pricingSoraV2 bills by duration + resolution multiplier.
func pricingSoraV2(req *relaycommon.TaskSubmitReq) map[string]float64 {
	duration := 4
	if req.Duration > 0 {
		duration = req.Duration
	}
	sizeRatio := 1.0
	if req.Size == "1792x1024" || req.Size == "1024x1792" {
		sizeRatio = 1.666667
	}
	return map[string]float64{
		"seconds": float64(duration),
		"size":    sizeRatio,
	}
}

// getPricingFunc returns the PricingFunc for a given model name.
// It first tries exact match, then prefix match (longest prefix wins).
func getPricingFunc(modelName string) PricingFunc {
	// Exact match first
	if fn, ok := pricingRegistry[modelName]; ok {
		return fn
	}
	// Prefix match: find longest matching prefix
	var bestFn PricingFunc
	bestLen := 0
	for prefix, fn := range pricingRegistry {
		if len(prefix) > bestLen && len(modelName) >= len(prefix) && modelName[:len(prefix)] == prefix {
			bestFn = fn
			bestLen = len(prefix)
		}
	}
	if bestFn != nil {
		return bestFn
	}
	// Default: per-second billing
	return pricingPerSecond
}

// ============================
// TaskAdaptor implementation
// ============================

var (
	ModelList = []string{
		"cogvideox", "cogvideox-2", "cogvideox-3",
		"sora-2", "sora-2-pro",
	}
	ChannelName = "zhipu_video"

	submitEndpoint = "/api/paas/v4/videos/generations"
	fetchEndpoint  = "/api/paas/v4/async-result"
)

type TaskAdaptor struct {
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	if a.baseURL == "" {
		a.baseURL = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeZhipu_v4]
	}
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	taskErr = relaycommon.ValidateBasicTaskRequest(c, info, channelconstant.TaskActionGenerate)
	if taskErr != nil {
		return
	}

	// Apply model-specific pricing
	var req relaycommon.TaskSubmitReq
	if v, exists := c.Get("task_request"); exists {
		if r, ok := v.(relaycommon.TaskSubmitReq); ok {
			req = r
		}
	}
	pricingFn := getPricingFunc(req.Model)
	info.PriceData.OtherRatios = pricingFn(&req)

	return
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s%s", a.baseURL, submitEndpoint), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, fmt.Errorf("invalid request type in context")
	}

	body := a.convertToRequestPayload(&req)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// TODO: debug only, remove after testing
	logger.LogInfo(c, fmt.Sprintf("zhipu video request body: %s", string(data)))

	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// TODO: debug only, remove after testing
	logger.LogInfo(c, fmt.Sprintf("zhipu video response body: %s", string(responseBody)))

	var zResp zhipuVideoSubmitResponse
	if err := json.Unmarshal(responseBody, &zResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if zResp.ID == "" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("zhipu video api error: empty task id, body: %s", responseBody), "task_failed", http.StatusBadRequest)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = zResp.ID
	ov.TaskID = zResp.ID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return zResp.ID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := fmt.Sprintf("%s%s/%s", baseUrl, fetchEndpoint, taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var zResp zhipuVideoFetchResponse
	if err := json.Unmarshal(respBody, &zResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal zhipu video task result failed")
	}

	taskInfo := &relaycommon.TaskInfo{
		TaskID: zResp.ID,
	}

	switch zResp.TaskStatus {
	case "PROCESSING":
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = "50%"
	case "SUCCESS":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if len(zResp.VideoResult) > 0 {
			taskInfo.Url = zResp.VideoResult[0].URL
		}
	case "FAIL":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = "zhipu video generation failed"
	default:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = "30%"
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var zResp zhipuVideoFetchResponse
	// Try to parse as fetch response first; if it fails, try submit response
	if err := json.Unmarshal(originTask.Data, &zResp); err != nil {
		// Might be the original submit response, still build a basic OpenAI video
		var submitResp zhipuVideoSubmitResponse
		if err2 := json.Unmarshal(originTask.Data, &submitResp); err2 != nil {
			return nil, errors.Wrap(err, "unmarshal zhipu task data failed")
		}
	}

	openAIVideo := originTask.ToOpenAIVideo()

	if len(zResp.VideoResult) > 0 {
		video := zResp.VideoResult[0]
		if video.URL != "" {
			openAIVideo.SetMetadata("url", video.URL)
		}
		if video.CoverImageURL != "" {
			openAIVideo.SetMetadata("cover_image_url", video.CoverImageURL)
		}
	}

	jsonData, err := common.Marshal(openAIVideo)
	if err != nil {
		return nil, errors.Wrap(err, "marshal openai video failed")
	}

	return jsonData, nil
}

// ============================
// Helpers
// ============================

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) *zhipuVideoRequest {
	body := &zhipuVideoRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
	}

	if body.Model == "" {
		body.Model = "cogvideox-3"
	}

	// Handle image input
	if len(req.Images) > 1 {
		// Multiple images: first frame + last frame (for start-end models)
		body.ImageURL = req.Images
	} else if len(req.Images) == 1 {
		body.ImageURL = req.Images[0]
	} else if req.Image != "" {
		body.ImageURL = req.Image
	}

	// Handle size
	if req.Size != "" {
		body.Size = req.Size
	}

	// Handle duration
	if req.Duration > 0 {
		body.Duration = req.Duration
	}

	// Apply metadata overrides (quality, with_audio, watermark_enabled, fps, etc.)
	if req.Metadata != nil {
		if v, ok := req.Metadata["quality"].(string); ok {
			body.Quality = v
		}
		if v, ok := req.Metadata["with_audio"].(bool); ok {
			body.WithAudio = &v
		}
		if v, ok := req.Metadata["watermark_enabled"].(bool); ok {
			body.WatermarkEnabled = &v
		}
		if v, ok := req.Metadata["fps"]; ok {
			switch fv := v.(type) {
			case float64:
				body.FPS = int(fv)
			case int:
				body.FPS = fv
			}
		}
		if v, ok := req.Metadata["request_id"].(string); ok {
			body.RequestID = v
		}
		if v, ok := req.Metadata["user_id"].(string); ok {
			body.UserID = v
		}
	}

	return body
}
