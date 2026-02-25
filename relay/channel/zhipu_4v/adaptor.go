package zhipu_4v

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	channelconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	return req, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	klingReq := map[string]any{
		"model": request.Model,
	}
	// text: prefer Text (Kling native), fallback to Input (OpenAI compat)
	if request.Text != "" {
		klingReq["text"] = request.Text
	} else {
		klingReq["text"] = request.Input
	}
	// voice_id: prefer VoiceID (Kling native), fallback to Voice (OpenAI compat)
	if request.VoiceID != "" {
		klingReq["voice_id"] = request.VoiceID
	} else if request.Voice != "" {
		klingReq["voice_id"] = request.Voice
	}
	// voice_language
	if request.VoiceLanguage != "" {
		klingReq["voice_language"] = request.VoiceLanguage
	}
	// voice_speed: prefer VoiceSpeed (Kling native), fallback to Speed (OpenAI compat)
	if request.VoiceSpeed > 0 {
		klingReq["voice_speed"] = request.VoiceSpeed
	} else if request.Speed > 0 {
		klingReq["voice_speed"] = request.Speed
	}
	data, err := json.Marshal(klingReq)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	zhipuReq := zhipuImageRequest{
		Model:   request.Model,
		Prompt:  request.Prompt,
		N:       request.N,
		Quality: request.Quality,
		Size:    request.Size,
	}
	if len(request.WatermarkEnabled) > 0 {
		var v bool
		if err := json.Unmarshal(request.WatermarkEnabled, &v); err == nil {
			zhipuReq.WatermarkEnabled = &v
		}
	}
	if len(request.UserId) > 0 {
		var v string
		if err := json.Unmarshal(request.UserId, &v); err == nil {
			zhipuReq.UserID = v
		}
	}

	// Populate struct fields from Extra (only fields defined in zhipuImageRequest are accepted)
	if len(request.Extra) > 0 {
		extraJSON, err := json.Marshal(request.Extra)
		if err == nil {
			json.Unmarshal(extraJSON, &zhipuReq)
		}
	}

	// TODO: debug only, remove after testing
	if debugJSON, err := json.Marshal(zhipuReq); err == nil {
		logger.LogInfo(c, fmt.Sprintf("zhipu image request body: %s", string(debugJSON)))
	}

	return zhipuReq, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := info.ChannelBaseUrl
	if baseURL == "" {
		baseURL = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeZhipu_v4]
	}
	specialPlan, hasSpecialPlan := channelconstant.ChannelSpecialBases[baseURL]

	switch info.RelayFormat {
	case types.RelayFormatClaude:
		if hasSpecialPlan && specialPlan.ClaudeBaseURL != "" {
			return fmt.Sprintf("%s/v1/messages", specialPlan.ClaudeBaseURL), nil
		}
		return fmt.Sprintf("%s/api/anthropic/v1/messages", baseURL), nil
	default:
		switch info.RelayMode {
		case relayconstant.RelayModeAudioSpeech:
			return fmt.Sprintf("%s/api/paas/v4/audio/tts", baseURL), nil
		case relayconstant.RelayModeElementCreate:
			return fmt.Sprintf("%s/api/paas/v4/images/custom-elements", baseURL), nil
		case relayconstant.RelayModeIdentifyFace:
			return fmt.Sprintf("%s/api/paas/v4/videos/identify-face", baseURL), nil
		case relayconstant.RelayModeEmbeddings:
			if hasSpecialPlan && specialPlan.OpenAIBaseURL != "" {
				return fmt.Sprintf("%s/embeddings", specialPlan.OpenAIBaseURL), nil
			}
			return fmt.Sprintf("%s/api/paas/v4/embeddings", baseURL), nil
		case relayconstant.RelayModeImagesGenerations:
			return fmt.Sprintf("%s/api/paas/v4/images/generations", baseURL), nil
		default:
			if hasSpecialPlan && specialPlan.OpenAIBaseURL != "" {
				return fmt.Sprintf("%s/chat/completions", specialPlan.OpenAIBaseURL), nil
			}
			return fmt.Sprintf("%s/api/paas/v4/chat/completions", baseURL), nil
		}
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if request.TopP >= 1 {
		request.TopP = 0.99
	}
	return requestOpenAI2Zhipu(*request), nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		adaptor := claude.Adaptor{}
		return adaptor.DoResponse(c, resp, info)
	default:
		if info.RelayMode == relayconstant.RelayModeAudioSpeech ||
			info.RelayMode == relayconstant.RelayModeElementCreate ||
			info.RelayMode == relayconstant.RelayModeIdentifyFace {
			return zhipu4vTTSHandler(c, resp, info)
		}
		if info.RelayMode == relayconstant.RelayModeImagesGenerations {
			return zhipu4vImageHandler(c, resp, info)
		}
		adaptor := openai.Adaptor{}
		return adaptor.DoResponse(c, resp, info)
	}
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

func zhipu4vTTSHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	// Forward JSON response as-is to client
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(responseBody)

	usage := &dto.Usage{}
	usage.PromptTokens = info.GetEstimatePromptTokens()
	usage.TotalTokens = usage.PromptTokens
	return usage, nil
}
