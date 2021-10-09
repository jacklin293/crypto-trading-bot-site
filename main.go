package main

import (
	"crypto-trading-bot-api/controller"
	"crypto-trading-bot-engine/util/logger"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// Read config
	loadConfig()

	// New logger
	l := logger.NewLogger(viper.GetString("ENV"), viper.GetString("LOG_PATH"))
	if viper.GetString("ENV") == "prod" {
		// Write access logs into file
		f, err := os.OpenFile(viper.GetString("GIN_LOG_PATH"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			l.Fatalf("failed to create file for gin log, err: %v", err)
		}
		gin.DefaultWriter = io.MultiWriter(f)
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	controller := controller.InitController(l)
	setRouter(r, controller)
	r.Run(fmt.Sprintf(":%s", viper.GetString("HTTP_PORT")))
}

func loadConfig() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	fmt.Printf("Load config (ENV: %s)\n", viper.Get("ENV"))
}
