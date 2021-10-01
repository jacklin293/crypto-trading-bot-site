package controller

import (
	"crypto-trading-bot-engine/db"
	"log"

	"github.com/spf13/viper"
)

type Controller struct {
	Strategy *Strategy
}

func InitController() *Controller {
	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	// Strategy
	strategy := &Strategy{
		db: db,
	}

	return &Controller{
		Strategy: strategy,
	}
}
