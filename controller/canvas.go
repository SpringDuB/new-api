package controller

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CanvasAuthorize(c *gin.Context) {
	setAuthNoStore(c)
	if _, _, enabled := common.StolasCanvasConfiguration(); !enabled {
		writeCanvasError(c, service.ErrCanvasDisabled)
		return
	}
	raw, ok := dashboardBearer(c.GetHeader("Authorization"))
	identity, err := service.ParseAccessToken(raw)
	if !ok || err != nil {
		writeCanvasError(c, service.ErrAuthTokenInvalid)
		return
	}
	var request service.CanvasAuthorizeRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeCanvasError(c, service.ErrCanvasAuthorization)
		return
	}
	redirect, err := service.AuthorizeCanvas(identity, request)
	if err != nil {
		writeCanvasError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"redirect_url": redirect})
}

func CanvasExchange(c *gin.Context) {
	setAuthNoStore(c)
	_, _, enabled := common.StolasCanvasConfiguration()
	if !enabled {
		writeCanvasError(c, service.ErrCanvasDisabled)
		return
	}
	bearer, ok := dashboardBearer(c.GetHeader("Authorization"))
	if !ok || !common.IsStolasCanvasClient(bearer) {
		writeCanvasError(c, service.ErrAuthTokenInvalid)
		return
	}
	var request service.CanvasExchangeRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeCanvasError(c, service.ErrCanvasAuthorization)
		return
	}
	result, err := service.ExchangeCanvas(request)
	if err != nil {
		writeCanvasError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func CanvasIdentity(c *gin.Context) {
	setAuthNoStore(c)
	if !common.IsStolasCanvasClient(c.GetHeader("X-Stolas-Client-Secret")) {
		writeCanvasError(c, service.ErrAuthTokenInvalid)
		return
	}
	identity, err := service.CanvasIdentity(c.GetInt("id"), c.GetInt("token_id"))
	if err != nil {
		writeCanvasError(c, err)
		return
	}
	common.ApiSuccess(c, identity)
}

func writeCanvasError(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "Operation failed"
	switch {
	case errors.Is(err, service.ErrCanvasDisabled):
		status, message = http.StatusServiceUnavailable, "Service unavailable"
	case errors.Is(err, service.ErrAuthTokenInvalid), errors.Is(err, model.ErrUserSessionInactive), errors.Is(err, gorm.ErrRecordNotFound):
		status, message = http.StatusUnauthorized, "Unauthorized"
	case errors.Is(err, model.ErrCanvasDefaultTokenUnavailable):
		status, message = http.StatusForbidden, "Token is invalid or has expired"
	case errors.Is(err, service.ErrCanvasAuthorization), errors.Is(err, model.ErrAuthFlowInvalid), errors.Is(err, model.ErrAuthFlowExpired), errors.Is(err, model.ErrAuthFlowConsumed):
		status, message = http.StatusBadRequest, "Invalid parameters"
	}
	c.JSON(status, gin.H{"success": false, "message": message})
}
