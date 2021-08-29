package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	setRouter(r)
	r.Run(":3000")
}
