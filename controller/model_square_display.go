package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetModelSquareDisplay(c *gin.Context) {
	common.ApiSuccessT(c, "", model.GetModelSquareDisplay())
}

func UpdateModelSquareDisplay(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
	var display model.ModelSquareDisplay
	if err := common.DecodeJson(c.Request.Body, &display); err != nil {
		common.ApiErrorT(c, "Invalid model square display settings")
		return
	}
	if err := display.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	encoded, err := common.Marshal(display)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOptionsBulk(map[string]string{model.ModelSquareDisplayOption: string(encoded)}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccessT(c, "Model square display saved", model.GetModelSquareDisplay())
}
