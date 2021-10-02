package controller

import (
	"crypto-trading-bot-engine/db"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Controller struct {
	db *db.DB
}

func InitController() *Controller {
	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	return &Controller{
		db: db,
	}
}

func failJSON(c *gin.Context, caller string, err error) {
	log.Printf("[ERROR] %s err: %s", caller, err.Error())
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   err.Error(),
	})
}

// NOTE intentionally provide vague for security purpose
func failJSONWithVagueError(c *gin.Context, caller string, err error) {
	log.Printf("[ERROR] %s err: %s", caller, err.Error())
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   "Internal error",
	})
}
