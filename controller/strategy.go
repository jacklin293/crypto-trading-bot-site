package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/exchange"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type Strategy struct {
	db *db.DB
}

// for template
type StrategyTmpl struct {
	Exchange       string
	SymbolPart1    string
	SymbolPart2    string
	Side           int64
	Leverage       string
	Margin         string
	Enabled        int64
	PositionStatus int64
	EntryPrice     string
	BoughtPrice    string
	TakeProfit     string
	StopLoss       string
}

func (s *Strategy) Index(c *gin.Context) {
	userUuid := "a8d59df4-47aa-4631-bbbc-42d4bb56d786" // FIXME
	exchangeName := "ftx"                              // FIXME
	user, err := s.db.GetUserByUuid(userUuid)
	if err != nil {
		log.Println("strategy controller err: ", err)
		c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
		return
	}

	// New exchange
	exchange, err := exchange.NewExchange(exchangeName, user.ExchangeApiInfo)
	if err != nil {
		log.Println("strategy controller err: ", err)
		c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
		return
	}

	// Get account info from exchange
	// NOTE Dont block if ftx api server is down
	var errMsg string
	accountInfo, err := exchange.GetAccountInfo()
	if err != nil {
		log.Println("strategy controller err: ", err)
		errMsg = "FTX API server is down"
	}

	// Get user data
	css, _, err := s.db.GetContractStrategiesByUser(userUuid)
	if err != nil {
		log.Println("strategy controller err: ", err)
		c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
		return
	}

	var strategyTmpls []StrategyTmpl
	for _, cs := range css {
		var st StrategyTmpl

		// Split symbol into 2 parts
		symbol := strings.Split(cs.Symbol, "-")

		// (position status: 1) Get entry price if position has been opened
		if len(cs.ExchangeOrdersDetails) != 0 {
			entryOrder, ok := cs.ExchangeOrdersDetails["entry_order"].(map[string]interface{})
			if ok {
				// position will show this price
				st.BoughtPrice = entryOrder["price"].(string)
			}
		}

		// entry price, stop-loss and take-profit
		if len(cs.Params) != 0 {
			contract, err := contract.NewContract(order.Side(cs.Side), cs.Params)
			if err != nil {
				log.Println("strategy controller err: ", err)
				c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
				return
			}
			// This doesn't matter for position
			st.EntryPrice = contract.EntryOrder.GetTrigger().GetPrice(time.Now()).Truncate(5).String()

			if contract.StopLossOrder != nil {
				// If entry_type is baseline, stop-loss trigger will be filled after entry order triggered
				stopLossTrigger := contract.StopLossOrder.GetTrigger()
				if stopLossTrigger != nil {
					st.StopLoss = stopLossTrigger.GetPrice(time.Now()).String()
					fmt.Println(st.StopLoss)
				}
			}

			if contract.TakeProfitOrder != nil {
				st.TakeProfit = contract.TakeProfitOrder.GetTrigger().GetPrice(time.Now()).String()
			}
		}

		if len(accountInfo) > 0 {
			st.Leverage = cs.Margin.Div(accountInfo["collateral"].(decimal.Decimal)).StringFixed(1)
		}

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

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":      "Strategy List",
		"strategies": strategyTmpls,
		"error":      errMsg,
	})
}
