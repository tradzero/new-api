package dto

import (
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type IdentifyFaceRequest struct {
	Model    string `json:"model"`
	VideoID  string `json:"video_id,omitempty"`
	VideoURL string `json:"video_url,omitempty"`
}

func (r *IdentifyFaceRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		CombineText: r.Model,
		TokenType:   types.TokenTypeTextNumber,
	}
}

func (r *IdentifyFaceRequest) IsStream(_ *gin.Context) bool {
	return false
}

func (r *IdentifyFaceRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
