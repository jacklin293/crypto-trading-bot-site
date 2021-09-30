package main

import (
	"crypto-trading-bot-api/controller"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func init() {
	// Read config
	loadConfig()
}

func main() {
	r := gin.Default()
	controller := controller.InitController()
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
