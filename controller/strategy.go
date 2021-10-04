package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/exchange"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

// for template
type StrategyTmpl struct {
	Exchange       string
	Symbol         string
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

func (ctl *Controller) StrategyList(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	var errMsg string
	success := c.Query("success")

	userCookie := ctl.getUserData(c)

	// Get exchange account info
	accountInfo, err := ctl.getExchangeAccountInfo(c)
	if err != nil {
		errMsg = "FTX API server is down"
	}

	// Get user data
	css, _, err := ctl.db.GetContractStrategiesByUser(userCookie.Uuid)
	if err != nil {
		log.Println("strategy controller err: ", err)
		c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
		return
	}

	symbolMap := make(map[string]bool)
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
				}
			}

			if contract.TakeProfitOrder != nil {
				st.TakeProfit = contract.TakeProfitOrder.GetTrigger().GetPrice(time.Now()).String()
			}
		}

		if len(accountInfo) > 0 {
			st.Leverage = cs.Margin.Div(accountInfo["collateral"].(decimal.Decimal)).StringFixed(1)
		}

		st.Exchange = cs.Exchange
		st.Symbol = cs.Symbol
		st.SymbolPart1 = symbol[0]
		st.SymbolPart2 = symbol[1]
		st.Side = cs.Side
		st.Margin = cs.Margin.String()
		st.Enabled = cs.Enabled
		st.PositionStatus = cs.PositionStatus
		strategyTmpls = append(strategyTmpls, st)

		// Prepare symbols array for js
		symbolMap[cs.Symbol] = true
	}

	// Prepare symbols array for js
	var symbols []string
	for key, _ := range symbolMap {
		symbols = append(symbols, key)
	}

	c.HTML(http.StatusOK, "strategy_list.html", gin.H{
		"symbols":    symbols,
		"strategies": strategyTmpls,
		"error":      errMsg,
		"loggedIn":   true,
		"success":    success,
	})
}

func (ctl *Controller) StrategyNewBaseline(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	var errMsg string

	// Get symbols
	symbols, _, err := ctl.db.GetEnabledContractSymbols("FTX")
	if err != nil {
		c.HTML(http.StatusOK, "strategy_new_baseline.html", gin.H{"error": "Symbols not found"})
		return
	}

	var collateral, leverage, totalMargin, availableMargin decimal.Decimal
	accountInfo, err := ctl.getExchangeAccountInfo(c)
	if err != nil {
		errMsg = "FTX API server is down"
	} else {
		collateral = accountInfo["collateral"].(decimal.Decimal)
		leverage = accountInfo["leverage"].(decimal.Decimal)
		totalMargin = collateral.Mul(leverage)
		availableMargin = accountInfo["free_collateral"].(decimal.Decimal).Mul(leverage)
	}

	c.HTML(http.StatusOK, "strategy_new_baseline.html", gin.H{
		"error":           errMsg,
		"loggedIn":        true,
		"symbols":         symbols,
		"collateral":      collateral.StringFixed(1),
		"leverage":        leverage.StringFixed(0),
		"totalMargin":     totalMargin.StringFixed(1),
		"availableMargin": availableMargin.StringFixed(1),
	})
}

func (ctl *Controller) StrategyCreate(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	// Validate symbols
	symbol := c.PostForm("symbol")
	symbolrows, _, err := ctl.db.GetEnabledContractSymbols("FTX")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error: symbols not found"})
		return
	}
	matched := false
	for _, symbolRow := range symbolrows {
		if symbolRow.Name == symbol {
			matched = true
		}
	}
	if !matched {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is invalid"})
		return
	}

	// Validate side
	sideString := c.PostForm("side")
	side, err := strconv.ParseInt(sideString, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side is invalid"})
		return
	}

	// Validate margin
	margin, err := decimal.NewFromString(c.PostForm("margin"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "margin is invalid"})
		return
	}

	// Stop-loss or take-profit enabled
	stopLossEnabled := c.PostForm("stop_loss[enabled]")
	takeProfitEnabled := c.PostForm("take_profit[enabled]")

	// Convert params
	params, err := ctl.convertBaselineStrategyParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare contract params
	contractParams := map[string]interface{}{
		"entry_type": c.PostForm("entry_type"),
		"entry_order": map[string]interface{}{
			"baseline_trigger": map[string]interface{}{
				"trigger_type": c.PostForm("entry[trigger_type]"),
				"operator":     c.PostForm("entry[operator]"),
				"time_1":       params["time_1"].(time.Time).Format(time.RFC3339),
				"price_1":      c.PostForm("entry[price_1]"),
				"time_2":       params["time_2"].(time.Time).Format(time.RFC3339),
				"price_2":      c.PostForm("entry[price_2]"),
			},
			"baseline_offset_percent": params["baseline_offset_percent"].(float64),
			"flip_operator_enabled":   params["flip_operator_enabled"].(bool),
		},
	}
	if stopLossEnabled == "1" {
		contractParams["stop_loss_order"] = map[string]interface{}{
			"loss_tolerance_percent":        params["loss_tolerance_percent"].(float64),
			"baseline_readjustment_enabled": params["baseline_readjustment_enabled"].(bool),
		}
	}
	if takeProfitEnabled == "1" {
		contractParams["take_profit_order"] = map[string]interface{}{
			"trigger": map[string]interface{}{
				"trigger_type": c.PostForm("take_profit[trigger_type]"),
				"operator":     c.PostForm("take_profit[operator]"),
				"price":        c.PostForm("take_profit[price]"),
			},
		}
	}

	// Validate contract params
	_, err = contract.NewContract(order.Side(side), contractParams)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create strategy
	userCookie := ctl.getUserData(c)
	strategy := db.ContractStrategy{
		Uuid:                  uuid.New().String(),
		UserUuid:              userCookie.Uuid,
		Symbol:                symbol,
		Margin:                margin,
		Side:                  side,
		Params:                contractParams,
		Enabled:               0,
		PositionStatus:        0,
		Exchange:              "FTX",
		ExchangeOrdersDetails: datatypes.JSONMap{},
	}
	insertId, count, err := ctl.db.CreateContractStrategy(strategy)
	if err != nil {
		log.Println("[ERROR] StrategyCreate db err: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}
	if insertId == 0 && count == 0 {
		log.Println("[ERROR] StrategyCreate insert id or count is 0")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
	return
}

func (ctl *Controller) StrategyNewLimit(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	var errMsg string
	c.HTML(http.StatusOK, "strategy_new_limit.html", gin.H{
		"error":    errMsg,
		"loggedIn": true,
	})
}

func (ctl *Controller) getExchangeAccountInfo(c *gin.Context) (accountInfo map[string]interface{}, err error) {
	userCookie := ctl.getUserData(c)
	user, err := ctl.db.GetUserByUuid(userCookie.Uuid)
	if err != nil {
		log.Println("strategy controller err: ", err)
		c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
		return
	}

	// New exchange
	exchange, err := exchange.NewExchange("ftx", user.ExchangeApiInfo)
	if err != nil {
		log.Println("strategy controller err: ", err)
		c.HTML(http.StatusOK, "index.html", gin.H{"error": "Internal Error"})
		return
	}

	// Get account info from exchange
	// NOTE Don't block if ftx api server is down
	accountInfo, err = exchange.GetAccountInfo()
	if err != nil {
		log.Println("strategy controller err: ", err)
	}
	return
}

func (ctl *Controller) convertBaselineStrategyParams(c *gin.Context) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// time 1
	time1 := c.PostForm("entry[time_1]")
	data["time_1"] = time.Now() // set default to avoid panic
	if time1 == "" {
		return data, errors.New("time_1 is missing")
	}
	t, err := time.Parse("2006-01-02 15:04", time1)
	if err != nil {
		return data, errors.New("time_1 is invalid")
	}
	data["time_1"] = t

	// time 2
	time2 := c.PostForm("entry[time_2]")
	data["time_2"] = time.Now()
	if time2 == "" {
		return data, errors.New("time_2 is missing")
	}
	t, err = time.Parse("2006-01-02 15:04", time2)
	if err != nil {
		return data, errors.New("time_2 is invalid")
	}
	data["time_2"] = t

	// baseline_offset_percent
	percent := c.PostForm("entry[baseline_offset_percent]")
	data["baseline_offset_percent"] = float64(0.0)
	if percent == "" {
		return data, errors.New("baseline_offset_percent is invalid")
	}
	p, err := strconv.ParseFloat(percent, 64)
	if err != nil {
		return data, errors.New("baseline_offset_percent is invalid")
	}
	data["baseline_offset_percent"] = float64(int64(p/100*10000)) / 10000 // convert to percent first, then fix float64

	// flip_operator_enabled
	enabled := c.PostForm("entry[flip_operator_enabled]")
	data["flip_operator_enabled"] = false
	if enabled == "" {
		return data, errors.New("flip_operator_enabled is invalid")
	}
	if enabled != "1" && enabled != "0" {
		return data, errors.New("flip_operator_enabled is invalid")
	}
	if enabled == "1" {
		data["flip_operator_enabled"] = true
	}
	if enabled == "0" {
		data["flip_operator_enabled"] = false
	}

	// stop loss enabled
	stopLossEnabled := c.PostForm("stop_loss[enabled]")

	// loss_tolerance_percent
	if stopLossEnabled == "1" {
		percent = c.PostForm("stop_loss[loss_tolerance_percent]")
		data["loss_tolerance_percent"] = float64(0.0)
		if percent == "" {
			return data, errors.New("loss_tolerance_percent is invalid")
		}
		p, err = strconv.ParseFloat(percent, 64)
		if err != nil {
			return data, errors.New("loss_tolerance_percent is invalid")
		}
		data["loss_tolerance_percent"] = float64(int64(p/100*10000)) / 10000 // convert to percent first, then fix float64
	}

	// baseline_readjustment_enabled
	if stopLossEnabled == "1" {
		enabled = c.PostForm("stop_loss[baseline_readjustment_enabled]")
		data["baseline_readjustment_enabled"] = false
		if enabled == "" {
			return data, errors.New("baseline_readjustment_enabled is invalid")
		}
		if enabled != "1" && enabled != "0" {
			return data, errors.New("baseline_readjustment_enabled is invalid")
		}
		if enabled == "1" {
			data["baseline_readjustment_enabled"] = true
		}
		if enabled == "0" {
			data["baseline_readjustment_enabled"] = false
		}
	}

	return data, nil
}
