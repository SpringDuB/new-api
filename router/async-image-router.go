package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func SetAsyncImageRouter(router *gin.Engine) {
	router.POST(
		"/v1/images/tasks",
		middleware.RouteTag("relay"),
		middleware.SystemPerformanceCheck(),
		middleware.TokenAuth(),
		middleware.ModelRequestRateLimit(),
		controller.CreateAsyncImageTask,
	)
	router.GET(
		"/v1/images/tasks/:task_id",
		middleware.RouteTag("relay"),
		middleware.TokenAuth(),
		controller.GetAsyncImageTask,
	)
}
