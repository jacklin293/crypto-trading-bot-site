package controller

import (
	"crypto-trading-bot-api/model"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Strategy struct {
	db *model.DB
}

func (s *Strategy) Index(c *gin.Context) {
	css, _, err := s.db.GetContractStrategiesByUser("a8d59df4-47aa-4631-bbbc-42d4bb56d786")
	if err != nil {
		log.Println("err: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title":      "Strategy List",
		"strategies": css,
	})
}
