package main

import (
	"fmt"
	"log"

	"crypto-trading-bot-api/db"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// Read config
	loadConfig()

	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	_ = db // FIXME

	r := gin.Default()
	setRouter(r)
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
