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
	Content          any    `json:"content,omitempty"`
	// Mode             string `json:"mode,omitempty"`
	ImageURL         any    `json:"image_url,omitempty"`
	Quality          string `json:"quality,omitempty"`
	WithAudio        *bool  `json:"with_audio,omitempty"`
	WatermarkEnabled *bool  `json:"watermark_enabled,omitempty"`
	Size             string `json:"size,omitempty"`
	FPS              int    `json:"fps,omitempty"`
	Duration         int    `json:"duration,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
	// Common video generation params
	FirstFrameImage  string `json:"first_frame_image,omitempty"`
	LastFrameImage   string `json:"last_frame_image,omitempty"`
	
	AspectRatio        string `json:"aspect_ratio,omitempty"`
	NegativePrompt     string `json:"negative_prompt,omitempty"`
	PersonGeneration   string `json:"person_generation,omitempty"`
	SampleCount        int    `json:"sample_count,omitempty"`
	Seed               int    `json:"seed,omitempty"`
	ResizeMode         string `json:"resize_mode,omitempty"`
	CompressionQuality string `json:"compression_quality,omitempty"`
	GenerateAudio      *bool  `json:"generate_audio,omitempty"`
	ServiceTier        string `json:"service_tier,omitempty"`
	ExecutionExpiresAfter int    `json:"execution_expires_after,omitempty"`
	Resolution         string `json:"resolution,omitempty"`
	PromptOptimizer    *bool  `json:"prompt_optimizer,omitempty"`
	FastPretreatment   *bool  `json:"fast_pretreatment,omitempty"`
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
	Usage       struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
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

	// Veo generate family: per-second + audio multiplier (2x with audio)
	"veo-3.0-generate": makePricingVeo(2.0),
	"veo-3.1-generate": makePricingVeo(2.0),

	// Veo fast family: per-second + audio multiplier (1.5x with audio)
	"veo-3.0-fast-generate": makePricingVeo(1.5),
	"veo-3.1-fast-generate": makePricingVeo(1.5),

	// Kling standard family: per-5s + mode multiplier (pro=1.75x)
	// ModelPrice = std/5s price
	"kling-v1-6":       makePricingKling(1.75),
	"kling-multi-v1-6": makePricingKling(1.75),
	"kling-v2-1":       makePricingKling(1.75),

	// Kling master family: per-5s, no mode
	// ModelPrice = 5s price
	"kling-v2-master":   pricingKlingMaster,
	"kling-v2-1-master": pricingKlingMaster,

	// Kling turbo: per-5s + mode multiplier (pro=5/3x)
	// ModelPrice = std/5s price
	"kling-v2-5-turbo": makePricingKling(5.0 / 3.0),

	// Seedance family: token-based billing with service_tier and audio multipliers
	// ModelPrice = offline + no audio rate ($0.0006/kTokens)
	// Actual billing adjusted by total_tokens on task completion
	"doubao-seedance": pricingSeedance,

	// Minimax-hailuo-2.3-Fast: ModelPrice = 768P/6s price
	"minimax-hailuo-2.3-Fast": makePricingHailuo(map[string]float64{
		"768P:6":  1.0,
		"768P:10": 32.0 / 19.0, // $0.32/$0.19
		"1080P:6": 33.0 / 19.0, // $0.33/$0.19
	}),

	// Minimax-hailuo-2.3: ModelPrice = 768P/6s price
	"minimax-hailuo-2.3": makePricingHailuo(map[string]float64{
		"768P:6":  1.0,
		"768P:10": 2.0,  // $0.56/$0.28
		"1080P:6": 1.75, // $0.49/$0.28
	}),

	// Minimax-hailuo-02: ModelPrice = 768P/6s price
	"minimax-hailuo-02": makePricingHailuo(map[string]float64{
		"512P:6":  10.0 / 28.0, // $0.10/$0.28
		"512P:10": 15.0 / 28.0, // $0.15/$0.28
		"768P:6":  1.0,
		"768P:10": 2.0,  // $0.56/$0.28
		"1080P:6": 1.75, // $0.49/$0.28
	}),
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

// makePricingVeo returns a PricingFunc for Veo models.
// ModelPrice should be set to the per-second rate WITHOUT audio.
// audioRatio is the multiplier applied when with_audio is true.
//   - generate models: audioRatio=2.0 ($0.20/s → $0.40/s)
//   - fast models:     audioRatio=1.5 ($0.10/s → $0.15/s)
func makePricingVeo(audioRatio float64) PricingFunc {
	return func(req *relaycommon.TaskSubmitReq) map[string]float64 {
		duration := 5
		if req.Duration > 0 {
			duration = req.Duration
		}
		audio := 1.0
		if req.WithAudio != nil && *req.WithAudio {
			audio = audioRatio
		}
		count := 1.0
		if req.SampleCount > 1 {
			count = float64(req.SampleCount)
		}
		return map[string]float64{
			"seconds":      float64(duration),
			"audio":        audio,
			"sample_count": count,
		}
	}
}

// makePricingKling returns a PricingFunc for Kling models with std/pro mode.
// ModelPrice should be set to the std/5s price.
// proRatio is the multiplier applied when mode is "pro".
//   - v1-6/multi/v2-1: proRatio=1.75 ($0.28 → $0.49)
//   - v2-5-turbo:      proRatio=5/3  ($0.21 → $0.35)
func makePricingKling(proRatio float64) PricingFunc {
	return func(req *relaycommon.TaskSubmitReq) map[string]float64 {
		duration := 5
		if req.Duration > 0 {
			duration = req.Duration
		}
		modeRatio := 1.0
		if req.Mode == "pro" {
			modeRatio = proRatio
		}
		return map[string]float64{
			"seconds": float64(duration) / 5.0,
			"mode":    modeRatio,
		}
	}
}

// pricingKlingMaster bills by duration only, no mode distinction.
// ModelPrice should be set to the 5s price.
func pricingKlingMaster(req *relaycommon.TaskSubmitReq) map[string]float64 {
	duration := 5
	if req.Duration > 0 {
		duration = req.Duration
	}
	return map[string]float64{
		"seconds": float64(duration) / 5.0,
	}
}

// pricingSeedance handles billing for seedance models.
// Two independent multipliers:
//   - service_tier: "default"(online)=2x, "flex"(offline)=1x
//   - generate_audio: true=2x, false=1x
//
// ModelPrice should be set to the base rate (offline + no audio).
// Actual billing is adjusted by total_tokens on task completion.
func pricingSeedance(req *relaycommon.TaskSubmitReq) map[string]float64 {
	serviceRatio := 2.0 // default = online
	if req.ServiceTier == "flex" {
		serviceRatio = 1.0
	}
	audioRatio := 1.0
	if req.GenerateAudio != nil && *req.GenerateAudio {
		audioRatio = 2.0
	}
	return map[string]float64{
		"service_tier": serviceRatio,
		"audio":        audioRatio,
	}
}

// makePricingHailuo returns a PricingFunc for Minimax-Hailuo models.
// ModelPrice should be set to the 768P/6s price in model configuration.
// ratioTable maps "resolution:duration" keys to price ratios relative to ModelPrice.
func makePricingHailuo(ratioTable map[string]float64) PricingFunc {
	return func(req *relaycommon.TaskSubmitReq) map[string]float64 {
		resolution := req.Resolution
		if resolution == "" {
			resolution = "768P"
		}
		duration := 6
		if req.Duration > 0 {
			duration = req.Duration
		}
		key := fmt.Sprintf("%s:%d", resolution, duration)
		if ratio, ok := ratioTable[key]; ok {
			return map[string]float64{
				"price": ratio,
			}
		}
		return map[string]float64{
			"price": 1.0,
		}
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
		"veo-3.0-generate-001", "veo-3.0-fast-generate-001",
		"veo-3.1-generate-preview", "veo-3.1-fast-generate-preview",
		"doubao-seedance",
		"minimax-hailuo",
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
		// Extract usage for token-based billing (e.g. seedance models)
		if zResp.Usage.TotalTokens > 0 {
			taskInfo.CompletionTokens = zResp.Usage.CompletionTokens
			taskInfo.TotalTokens = zResp.Usage.TotalTokens
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
		Model:              req.Model,
		WithAudio:          req.WithAudio,
		GenerateAudio:      req.GenerateAudio,
		ServiceTier:        req.ServiceTier,
		RequestID:          req.RequestID,
		AspectRatio:        req.AspectRatio,
		NegativePrompt:     req.NegativePrompt,
		PersonGeneration:   req.PersonGeneration,
		SampleCount:        req.SampleCount,
		Seed:               req.Seed,
		ResizeMode:         req.ResizeMode,
		CompressionQuality: req.CompressionQuality,
		FirstFrameImage:    req.FirstFrameImage,
		LastFrameImage:     req.LastFrameImage,
		Resolution:         req.Resolution,
		PromptOptimizer:    req.PromptOptimizer,
		FastPretreatment:   req.FastPretreatment,
		Quality:            req.Quality,
		FPS:                req.FPS,
	}

	if body.Model == "" {
		body.Model = "cogvideox-3"
	}

	// Content array format (seedance): text + image_url objects in one array
	// Prompt + image format (cogvideox, veo, sora, etc.): separate fields
	if req.Content != nil {
		body.Content = req.Content
	} else {
		body.Prompt = req.Prompt

		// Handle image input: prefer image_url (direct passthrough), fallback to images/image
		if req.ImageURL != nil {
			body.ImageURL = req.ImageURL
		} else if len(req.Images) > 1 {
			body.ImageURL = req.Images
		} else if len(req.Images) == 1 {
			body.ImageURL = req.Images[0]
		} else if req.Image != "" {
			body.ImageURL = req.Image
		}
	}

	// Handle size
	if req.Size != "" {
		body.Size = req.Size
	}

	// Handle duration
	if req.Duration > 0 {
		body.Duration = req.Duration
	}

	// Apply metadata fallbacks (quality, watermark_enabled, fps, user_id)
	if req.Metadata != nil {
		if body.Quality == "" {
			if v, ok := req.Metadata["quality"].(string); ok {
				body.Quality = v
			}
		}
		if v, ok := req.Metadata["watermark_enabled"].(bool); ok {
			body.WatermarkEnabled = &v
		}
		if body.FPS == 0 {
			if v, ok := req.Metadata["fps"]; ok {
				switch fv := v.(type) {
				case float64:
					body.FPS = int(fv)
				case int:
					body.FPS = fv
				}
			}
		}
		if v, ok := req.Metadata["user_id"].(string); ok {
			body.UserID = v
		}
		if body.FirstFrameImage == "" {
			if v, ok := req.Metadata["first_frame_image"].(string); ok {
				body.FirstFrameImage = v
			}
		}
		if body.LastFrameImage == "" {
			if v, ok := req.Metadata["last_frame_image"].(string); ok {
				body.LastFrameImage = v
			}
		}
	}

	return body
}
