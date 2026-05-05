package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, Envelope{Code: status, Message: msg})
}
