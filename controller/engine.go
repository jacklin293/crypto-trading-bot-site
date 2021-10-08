package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (ctl *Controller) Engine(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	userCookie := ctl.getUserData(c)
	if userCookie.Role != 99 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	// ping
	var ping, status, list string
	body, err := ctl.makeRequestToEngine("/ping")
	if err != nil {
		ping = err.Error()
	} else {
		ping = string(body)
	}

	// status
	body, err = ctl.makeRequestToEngine("/status")
	if err != nil {
		status = err.Error()
	} else {
		status = string(body)
	}

	// status
	body, err = ctl.makeRequestToEngine("/list")
	if err != nil {
		list = err.Error()
	} else {
		list = string(body)
	}

	c.HTML(http.StatusOK, "engine.html", gin.H{
		"loggedIn": true,
		"role":     userCookie.Role,
		"ping":     ping,
		"status":   status,
		"list":     list,
	})
}
