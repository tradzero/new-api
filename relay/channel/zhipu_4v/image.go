package zhipu_4v

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type sequentialImageGenerationOptions struct {
    MaxImages      int    `json:"max_images,omitempty"`
    ResponseFormat string `json:"response_format,omitempty"`
    Watermark      *bool  `json:"watermark,omitempty"`
}


type zhipuImageRequest struct {
	Model            string `json:"model"`
	Prompt           string `json:"prompt"`
	Quality          string `json:"quality,omitempty"`
	Size             string `json:"size,omitempty"`
	Ratio            string `json:"ratio,omitempty"`
	WatermarkEnabled *bool  `json:"watermark_enabled,omitempty"`
	UserID           string `json:"user_id,omitempty"`
	Seed             int `json:"seed,omitempty"`
	SequentialImageGeneration string `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOptions *sequentialImageGenerationOptions `json:"sequential_image_generation_options,omitempty"`
}

type zhipuImageResponse struct {
	Created       *int64            `json:"created,omitempty"`
	Data          []zhipuImageData  `json:"data,omitempty"`
	ContentFilter any               `json:"content_filter,omitempty"`
	Usage         *dto.Usage        `json:"usage,omitempty"`
	Error         *zhipuImageError  `json:"error,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	ExtendParam   map[string]string `json:"extendParam,omitempty"`
}

type zhipuImageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type zhipuImageData struct {
	Url      string `json:"url,omitempty"`
	ImageUrl string `json:"image_url,omitempty"`
	B64Json  string `json:"b64_json,omitempty"`
	B64Image string `json:"b64_image,omitempty"`
}

type openAIImagePayload struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
	Usage   *dto.Usage        `json:"usage,omitempty"`
}

type openAIImageData struct {
	Url     string `json:"url,omitempty"`
	B64Json string `json:"b64_json,omitempty"`
}

func zhipu4vImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	// TODO: debug only, remove after testing
	// logger.LogInfo(c, fmt.Sprintf("zhipu image response body: %s", string(responseBody)))

	var zhipuResp zhipuImageResponse
	if err := common.Unmarshal(responseBody, &zhipuResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if zhipuResp.Error != nil && zhipuResp.Error.Message != "" {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: zhipuResp.Error.Message,
			Type:    "zhipu_image_error",
			Code:    zhipuResp.Error.Code,
		}, resp.StatusCode)
	}

	payload := openAIImagePayload{}
	if zhipuResp.Created != nil && *zhipuResp.Created != 0 {
		payload.Created = *zhipuResp.Created
	} else {
		payload.Created = info.StartTime.Unix()
	}
	// Determine response format: check top-level first, then fall back to sequential_image_generation_options
	responseFormat := ""
	if imageReq, ok := info.Request.(*dto.ImageRequest); ok {
		responseFormat = imageReq.ResponseFormat
		if responseFormat == "" {
			if val, ok := imageReq.Extra["sequential_image_generation_options"]; ok {
				var opts struct {
					ResponseFormat string `json:"response_format"`
				}
				if err := json.Unmarshal(val, &opts); err == nil {
					responseFormat = opts.ResponseFormat
				}
			}
		}
	}

	for _, data := range zhipuResp.Data {
		url := data.Url
		if url == "" {
			url = data.ImageUrl
		}

		var imageData openAIImageData

		// If response_format is "url" and we have an HTTP URL, return it directly
		if responseFormat == "url" && url != "" && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			imageData.Url = url
		} else {
			// Otherwise convert to base64
			var b64 string
			switch {
			case data.B64Json != "":
				b64 = data.B64Json
			case data.B64Image != "":
				b64 = data.B64Image
			case url != "" && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")):
				_, downloaded, err := service.GetImageFromUrl(url)
				if err != nil {
					logger.LogError(c, "zhipu_image_get_b64_failed: "+err.Error())
					continue
				}
				b64 = downloaded
			case url != "":
				// URL field contains raw base64 data
				b64 = url
			default:
				logger.LogWarn(c, "zhipu_image_missing_url")
				continue
			}

			if b64 == "" {
				logger.LogWarn(c, "zhipu_image_empty_b64")
				continue
			}
			imageData.B64Json = b64
		}

		payload.Data = append(payload.Data, imageData)
	}

	usage := &dto.Usage{}
	if zhipuResp.Usage != nil {
		usage = zhipuResp.Usage
		if usage.PromptTokens == 0 && usage.InputTokens != 0 {
			usage.PromptTokens = usage.InputTokens
		}
		if usage.CompletionTokens == 0 && usage.OutputTokens != 0 {
			usage.CompletionTokens = usage.OutputTokens
		}
		if usage.TotalTokens == 0 {
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		}
	}
	payload.Usage = usage

	// For fixed-price models, adjust ModelPrice to reflect actual image count
	// so that settlement refunds the difference if fewer images were generated
	if info.PriceData.UsePrice {
		actualCount := len(payload.Data)
		requestN := 0
		if imageReq, ok := info.Request.(*dto.ImageRequest); ok && imageReq.N > 0 {
			requestN = int(imageReq.N)
		}
		if requestN > 0 && actualCount < requestN {
			info.PriceData.ModelPrice = info.PriceData.ModelPrice / float64(requestN) * float64(actualCount)
		}
	}

	// avoid escaped characters in JSON response
	// response url format problem
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	jsonResp := bytes.TrimSuffix(buf.Bytes(), []byte("\n"))

	service.IOCopyBytesGracefully(c, resp, jsonResp)

	return usage, nil
}
