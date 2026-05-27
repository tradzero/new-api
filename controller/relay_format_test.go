package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayHandlerRoutesElementAndIdentifyFaceAwayFromTextHelper(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		path      string
		relayMode int
		format    types.RelayFormat
		request   dto.Request
	}{
		{
			name:      "element",
			path:      "/v1/elements",
			relayMode: relayconstant.RelayModeElementCreate,
			format:    types.RelayFormatElement,
			request: &dto.ElementRequest{
				Model:       "kling-image",
				ElementName: "test",
			},
		},
		{
			name:      "identify face",
			path:      "/v1/video/identify-face",
			relayMode: relayconstant.RelayModeIdentifyFace,
			format:    types.RelayFormatIdentifyFace,
			request: &dto.IdentifyFaceRequest{
				Model:   "kling-video",
				VideoID: "video-id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, tt.path, nil)
			c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeAIProxyLibrary)

			err := relayHandler(c, &relaycommon.RelayInfo{
				RelayMode:   tt.relayMode,
				RelayFormat: tt.format,
				Request:     tt.request,
			})

			require.Error(t, err)
			require.NotContains(t, err.Error(), "expected dto.GeneralOpenAIRequest")
			require.Contains(t, err.Error(), "invalid api type")
		})
	}
}
