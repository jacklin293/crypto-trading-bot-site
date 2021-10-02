package controller

import (
	"github.com/gin-gonic/gin"
)

func (ctl *Controller) Ping(c *gin.Context) {
	c.String(200, "pong")
}
