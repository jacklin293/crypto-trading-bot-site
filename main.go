package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Static("/assets", "assets")
	r.LoadHTMLGlob("templates/*")
	r.GET("/ping", ping)
	r.GET("/", index)
	r.Run(":3000") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
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
