package controller

import "github.com/gin-gonic/gin"

func (ctl *Controller) Releases(c *gin.Context) {
	c.HTML(200, "releases.html", gin.H{})
}
