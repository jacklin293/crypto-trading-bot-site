package controller

import (
	"crypto-trading-bot-api/model"
	"log"
	"net/http"
	"strings"

	// "github.com/jacklin293/crypto-trading-bot-main/strategy/contract"
	// "github.com/jacklin293/crypto-trading-bot-main/strategy/order"

	"github.com/gin-gonic/gin"
)

type Strategy struct {
	db *model.DB
}

// for template
type StrategyTmpl struct {
	Exchange       string
	SymbolPart1    string
	SymbolPart2    string
	Side           int64
	Margin         string
	Enabled        int64
	PositionStatus int64
	EntryPrice     string
	TakeProfit     string
	StopLoss       string
}

func (s *Strategy) Index(c *gin.Context) {
	css, _, err := s.db.GetContractStrategiesByUser("a8d59df4-47aa-4631-bbbc-42d4bb56d786")
	if err != nil {
		log.Println("GetContractStrategiesByUser , err: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	var strategyTmpls []StrategyTmpl
	for _, cs := range css {
		var st StrategyTmpl

		// NOTE FIXME support other exchange
		// Split symbol into 2 parts
		symbol := strings.Split(cs.Symbol, "-")

		// (position status: 1) Get entry price if position has been opened
		if len(cs.ExchangeOrdersDetails) != 0 {
			entryOrder, ok := cs.ExchangeOrdersDetails["entry_order"].(map[string]interface{})
			if ok {
				st.EntryPrice = entryOrder["price"].(string)
			}
		}

		// entry price, stop-loss and take-profit
		/*
			if len(cs.Params) {
				contract, err := contract.NewContract(order.Side(cs.Side), cs.Params)
				if err != nil {
					log.Println("NewContract err: ", err.Error())
				}
				st.EntryPrice = contract.EntryOrder.GetTrigger().GetPrice(time.Now())

				if contract.StopLoss != nil {
					st.StopLoss = contract.StopLoss.GetTrigger().GetPrice(time.Now())
				}

				if contract.TakeProfit != nil {
					st.TakeProfit = contract.TakeProfit.GetTrigger().GetPrice(time.Now())
				}
			}
		*/

		// (position status: 0)
		// TODO

		st.Exchange = cs.Exchange
		st.SymbolPart1 = symbol[0]
		st.SymbolPart2 = symbol[1]
		st.Side = cs.Side
		st.Margin = cs.Margin.String()
		st.Enabled = cs.Enabled
		st.PositionStatus = cs.PositionStatus

		strategyTmpls = append(strategyTmpls, st)
	}

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title":      "Strategy List",
		"strategies": strategyTmpls,
	})
}
