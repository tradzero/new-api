package dto

import (
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type ElementRequest struct {
	Model               string `json:"model"`

	// for kling element model
	ElementName         string `json:"element_name"`
	ElementDescription  string `json:"element_description,omitempty"`
	ElementFrontalImage string `json:"element_frontal_image,omitempty"`
	ElementReferList    any    `json:"element_refer_list,omitempty"`
}

func (r *ElementRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		CombineText: r.ElementName,
		TokenType:   types.TokenTypeTextNumber,
	}
}

func (r *ElementRequest) IsStream(_ *gin.Context) bool {
	return false
}

func (r *ElementRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
