package main

import (
	"crypto-trading-bot-api/controller"

	"github.com/gin-gonic/gin"
)

func setRouter(r *gin.Engine, c *controller.Controller) {

	// Static files
	r.Static("/assets", "assets")

	// Html template
	r.LoadHTMLGlob("view/*")

	// Health check
	r.GET("/ping", controller.Ping)

	// Home page
	r.GET("/", c.Strategy.Index)
}
