package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func setRouter(r *gin.Engine) {
	// Static files
	r.Static("/assets", "assets")

	// Html template
	r.LoadHTMLGlob("templates/*")

	// Health check
	r.GET("/ping", ping)

	// Home page
	r.GET("/", index)
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Main website",
	})
}
