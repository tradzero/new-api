package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestGenRelayInfoSupportsElementAndIdentifyFaceFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		path      string
		format    types.RelayFormat
		request   dto.Request
		relayMode int
	}{
		{
			name:   "element",
			path:   "/v1/elements",
			format: types.RelayFormatElement,
			request: &dto.ElementRequest{
				Model:       "kling-image",
				ElementName: "test",
			},
			relayMode: relayconstant.RelayModeElementCreate,
		},
		{
			name:   "identify face",
			path:   "/v1/video/identify-face",
			format: types.RelayFormatIdentifyFace,
			request: &dto.IdentifyFaceRequest{
				Model:   "kling-video",
				VideoID: "video-id",
			},
			relayMode: relayconstant.RelayModeIdentifyFace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, tt.path, nil)

			info, err := GenRelayInfo(c, tt.format, tt.request, nil)

			require.NoError(t, err)
			require.Equal(t, tt.format, info.RelayFormat)
			require.Equal(t, tt.relayMode, info.RelayMode)
			require.Same(t, tt.request, info.Request)
			require.Equal(t, []types.RelayFormat{tt.format}, info.RequestConversionChain)
		})
	}
}
