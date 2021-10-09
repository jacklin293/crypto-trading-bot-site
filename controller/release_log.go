package controller

import "github.com/gin-gonic/gin"

func (ctl *Controller) ReleaseLog(c *gin.Context) {
	c.HTML(200, "release_log.html", gin.H{
		"loggedIn": true,
		"role":     ctl.getUserData(c).Role,
	})
}
