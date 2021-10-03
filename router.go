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

	// Account
	r.GET("/login", c.LoginPage)
	r.POST("/login", c.LoginAPI)
	r.POST("/otp", c.OTP)
	r.GET("/logout", c.Logout)

	// strategy
	r.GET("/", c.StrategyList)
	r.GET("/strategy/new_baseline", c.StrategyNewBaseline)
	r.GET("/strategy/new_limit", c.StrategyNewLimit)
}
