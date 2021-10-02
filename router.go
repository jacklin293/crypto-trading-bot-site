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
	r.GET("/ping", c.Ping)

	// Login
	r.GET("/login", c.LoginGET)
	r.POST("/login", c.LoginPOST)
	r.POST("/otp", c.OTP)

	// Logout
	r.POST("/logout", c.LogoutPOST)

	// Home page
	r.GET("/", c.StrategyList)

}
