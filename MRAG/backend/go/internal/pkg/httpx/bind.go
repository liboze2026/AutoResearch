package httpx

import "github.com/gin-gonic/gin"

func MustBindJSON(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		Error(c, 400, err.Error())
		return false
	}
	return true
}
