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

	// Release log
	r.GET("/releases", c.Releases)

	// Account
	r.GET("/login", c.LoginPage)
	r.POST("/login", c.LoginAPI)
	r.POST("/otp", c.OTP)
	r.GET("/logout", c.Logout)

	// strategy
	r.GET("/", c.ListStrategies)
	r.GET("/strategy/new_trendline", c.NewStrategy)
	r.GET("/strategy/new_limit", c.NewStrategy)
	r.POST("/strategy", c.CreateStrategy)
	r.GET("/strategy/:uuid", c.ShowStrategy)
	r.DELETE("/strategy/:uuid", c.DeleteStrategy)

	// action
	r.GET("/action/enable_strategy/:uuid", c.EnableStrategy)
	r.GET("/action/disable_strategy/:uuid", c.DisableStrategy)
	r.GET("/action/reset_strategy/:uuid", c.ResetStrategy)
	r.GET("/action/share_strategy/:uuid", c.ShareStrategy) // TODO
	r.GET("/action/close_position/:uuid", c.ClosePosition)
}
