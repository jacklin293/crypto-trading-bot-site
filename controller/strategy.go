package controller

import (
	"bytes"
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/exchange"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"gorm.io/datatypes"
)

// for template
type StrategyTmpl struct {
	Uuid           string
	Exchange       string
	Symbol         string
	SymbolPart1    string
	SymbolPart2    string
	Side           int64
	Leverage       string
	Enabled        int64
	PositionStatus int64
	EntryPrice     string
	BoughtPrice    string
	TakeProfit     string
	StopLoss       string
	Comment        string
}

func (ctl *Controller) ListStrategies(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	// Allow other pages bring message to here and show on lsit page
	var errMsg string
	success := c.Query("success")

	// Get user from cookie
	userCookie := ctl.getUserData(c)

	// Get exchange account info
	accountInfo, err := ctl.getExchangeAccountInfo(c)
	if err != nil {
		errMsg = fmt.Sprintf("%s API server 無回應或 API Key 已失效", viper.GetString("DEFAULT_EXCHANGE"))
	}

	// Get user data
	css, _, err := ctl.db.GetContractStrategiesByUser(userCookie.Uuid)
	if err != nil {
		ctl.log.Println("strategy controller err: ", err)
		errMsg = "Internal error"
	}

	// For money and currency formatting
	ac := accounting.Accounting{Symbol: "$", Precision: 8}

	symbolMap := make(map[string]bool)
	var strategyTmpls []StrategyTmpl
	for _, cs := range css {
		var st StrategyTmpl

		// Split symbol into 2 parts
		symbol := strings.Split(cs.Symbol, "-")

		// (position status: 1) Get entry price if position has been opened
		var boughtPrice decimal.Decimal
		if len(cs.ExchangeOrdersDetails) != 0 {
			entryOrder, ok := cs.ExchangeOrdersDetails["entry_order"].(map[string]interface{})
			if ok {
				boughtPrice, err = decimal.NewFromString(entryOrder["price"].(string))
				if err != nil {
					ctl.log.Println("strategy controller - failed to convert entryOrder[price], err: ", err)
					errMsg = "Internal error"
					continue
				}
			}
		}

		// entry price, stop-loss and take-profit
		if len(cs.Params) != 0 {
			contract, err := contract.NewContract(order.Side(cs.Side), cs.Params)
			if err != nil {
				ctl.log.Println("strategy controller err: ", err)
				errMsg = "Internal error"
				continue
			}
			// This doesn't matter for position
			st.EntryPrice = ac.FormatMoneyDecimal(contract.EntryOrder.GetTrigger().GetPrice(time.Now()))

			if contract.StopLossOrder != nil {
				// If entry_type is trendline, stop-loss trigger will be filled after entry order triggered
				stopLossTrigger := contract.StopLossOrder.GetTrigger()
				if stopLossTrigger != nil {
					st.StopLoss = ac.FormatMoneyDecimal(stopLossTrigger.GetPrice(time.Now()))
				}
			}

			if contract.TakeProfitOrder != nil {
				st.TakeProfit = ac.FormatMoneyDecimal(contract.TakeProfitOrder.GetTrigger().GetPrice(time.Now()))
			}
		}

		if len(accountInfo) > 0 {
			st.Leverage = cs.Margin.Div(accountInfo["collateral"].(decimal.Decimal)).StringFixed(1)
		}

		st.Uuid = cs.Uuid
		st.Exchange = cs.Exchange
		st.Symbol = cs.Symbol
		st.SymbolPart1 = symbol[0]
		st.SymbolPart2 = symbol[1]
		st.Side = cs.Side
		st.Enabled = cs.Enabled
		st.PositionStatus = cs.PositionStatus
		st.BoughtPrice = ac.FormatMoneyDecimal(boughtPrice)
		st.Comment = cs.Comment
		strategyTmpls = append(strategyTmpls, st)

		// Prepare symbols array for js
		symbolMap[cs.Symbol] = true
	}

	// Prepare symbols array for js
	var symbols []string
	for key, _ := range symbolMap {
		symbols = append(symbols, key)
	}

	c.HTML(http.StatusOK, "list_strategies.html", gin.H{
		"loggedIn":   true,
		"role":       userCookie.Role,
		"symbols":    symbols,
		"strategies": strategyTmpls,
		"error":      errMsg,
		"success":    success,
	})
}

func (ctl *Controller) NewStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	var errMsg string

	// Get symbols
	symbols, _, err := ctl.db.GetEnabledContractSymbols(viper.GetString("DEFAULT_EXCHANGE"))
	if err != nil {
		c.HTML(http.StatusOK, "new_trendline_strategy.html", gin.H{"error": "Symbols not found"})
		return
	}

	var collateral, leverage, totalMargin, availableMargin decimal.Decimal
	accountInfo, err := ctl.getExchangeAccountInfo(c)
	if err != nil {
		errMsg = fmt.Sprintf("%s API server 無回應或 API Key 已失效", viper.GetString("DEFAULT_EXCHANGE"))
	} else {
		collateral = accountInfo["collateral"].(decimal.Decimal)
		leverage = accountInfo["leverage"].(decimal.Decimal)
		totalMargin = collateral.Mul(leverage)
		availableMargin = accountInfo["free_collateral"].(decimal.Decimal).Mul(leverage)
	}

	newStrategyHtml := "new_trendline_strategy.html"
	if c.FullPath() == "/strategy/new_limit" {
		newStrategyHtml = "new_limit_strategy.html"
	}

	c.HTML(http.StatusOK, newStrategyHtml, gin.H{
		"loggedIn":        true,
		"role":            ctl.getUserData(c).Role,
		"error":           errMsg,
		"symbols":         symbols,
		"collateral":      collateral.StringFixed(1),
		"leverage":        leverage.StringFixed(0),
		"totalMargin":     totalMargin.StringFixed(1),
		"availableMargin": availableMargin.StringFixed(1),
	})
}

func (ctl *Controller) CreateStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	// Validate symbols
	symbol := c.PostForm("symbol")
	symbolrows, _, err := ctl.db.GetEnabledContractSymbols(viper.GetString("DEFAULT_EXCHANGE"))
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

	// Convert params
	var contractParams map[string]interface{}
	switch c.PostForm("entry_type") {
	case "trendline":
		contractParams, err = ctl.processTrendlineContractParams(c)
	case "limit":
		contractParams, err = ctl.processLimitContractParams(c)
	default:
		err = errors.New("entry type not supported")
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
		Exchange:              viper.GetString("DEFAULT_EXCHANGE"),
		ExchangeOrdersDetails: datatypes.JSONMap{},
		Comment:               c.PostForm("comment"),
	}
	insertId, count, err := ctl.db.CreateContractStrategy(strategy)
	if err != nil {
		// Capture `Error 1406: Data too long for column 'comment' at row 1`
		if strings.Contains(err.Error(), "comment") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "註解字數過多"})
			return
		}

		ctl.log.Println("[ERROR] StrategyCreate db err: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}
	if insertId == 0 && count == 0 {
		ctl.log.Println("[ERROR] StrategyCreate insert id or count is 0")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
	return
}

func (ctl *Controller) ShowStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	userCookie := ctl.getUserData(c)
	uuid := c.Param("uuid")
	errMsg := ""

	// For money and currency formatting
	ac := accounting.Accounting{Symbol: "$", Precision: 8}

	// Get exchange account info
	accountInfo, err := ctl.getExchangeAccountInfo(c)
	if err != nil {
		errMsg = fmt.Sprintf("%s API server 無回應或 API Key 已失效", viper.GetString("DEFAULT_EXCHANGE"))
	}

	// Check permission
	strategy, err := ctl.db.GetContractStrategyByUuidByUser(uuid, userCookie.Uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	leverage := ""
	if len(accountInfo) > 0 {
		leverage = strategy.Margin.Div(accountInfo["collateral"].(decimal.Decimal)).StringFixed(1)
	}

	params, err := json.Marshal(strategy.Params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"}) // NOTE shouldn't use json, but it shouldn't happen
		return
	}
	params = bytes.Replace(params, []byte("\\u003c"), []byte("<"), -1)
	params = bytes.Replace(params, []byte("\\u003e"), []byte(">"), -1)

	ordersDetails := "(無)"
	if len(strategy.ExchangeOrdersDetails) > 0 {
		b, err := json.Marshal(strategy.ExchangeOrdersDetails)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"}) // NOTE shouldn't use json, but it shouldn't happen
			return
		}
		b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
		b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
		ordersDetails = string(b)
	}

	lastPositionAt := "(未開倉)"
	if strategy.LastPositionAt.Unix() > 0 {
		lastPositionAt = strategy.LastPositionAt.Format("2006-01-02 15:04:05")
	}

	comment := "(未填)"
	if strategy.Comment != "" {
		comment = strategy.Comment
	}

	c.HTML(http.StatusOK, "show_strategy.html", gin.H{
		"error":          errMsg,
		"loggedIn":       true,
		"role":           userCookie.Role,
		"strategy":       strategy,
		"margin":         ac.FormatMoneyDecimal(strategy.Margin),
		"leverage":       leverage,
		"params":         string(params),
		"comment":        comment,
		"ordersDetails":  ordersDetails,
		"lastPositionAt": lastPositionAt,
		"createdAt":      strategy.CreatedAt.Format("2006-01-02 15:04:05"),
		"updatedAt":      strategy.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

func (ctl *Controller) DeleteStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}
	userCookie := ctl.getUserData(c)
	uuid := c.Param("uuid")

	// Check permission
	strategy, err := ctl.db.GetContractStrategyByUuidByUser(uuid, userCookie.Uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	// Make sure the status has been disabed and position status is closed
	if strategy.Enabled != 0 || contract.Status(strategy.PositionStatus) != contract.CLOSED {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略未暫停或訂單狀態未結束"})
		return
	}

	// Make sure it's not tracked by engine
	if err = ctl.notBeingTrackedByEngine(c, uuid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete data
	result := ctl.db.GormDB.Delete(&strategy)
	if result.Error != nil {
		ctl.log.Println("[ERROR] failed to delete strategy, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) getExchangeAccountInfo(c *gin.Context) (accountInfo map[string]interface{}, err error) {
	ex, err := ctl.newExchange(c)
	if err != nil {
		return
	}

	// Get account info from exchange
	accountInfo, err = ex.GetAccountInfo()
	if err != nil {
		ctl.log.Printf("failed to get account info from %s, err: %s", viper.GetString("DEFAULT_EXCHANGE"), err.Error())
	}
	return
}

func (ctl *Controller) newExchange(c *gin.Context) (ex exchange.Exchanger, err error) {
	// Get user data
	userCookie := ctl.getUserData(c)
	user, err := ctl.db.GetUserByUuid(userCookie.Uuid)
	if err != nil {
		ctl.log.Printf("[ERROR] failed to get user by '%s', err: %v", userCookie.Uuid, err)
		err = errors.New("用戶不存在")
		return
	}

	if user.ExchangeApiKey == "" {
		err = errors.New("請先新增 API Key")
		return
	}

	// New exchange
	ex, err = exchange.NewExchange(viper.GetString("DEFAULT_EXCHANGE"), user.ExchangeApiKey)
	if err != nil {
		ctl.log.Println("[ERROR] failed to new exchange")
		err = errors.New("API Key 可能已失效, 請確認或重試一次")
		return
	}
	return
}

func (ctl *Controller) processLimitContractParams(c *gin.Context) (map[string]interface{}, error) {
	// Stop-loss or take-profit enabled
	stopLossEnabled := c.PostForm("stop_loss[enabled]")
	takeProfitEnabled := c.PostForm("take_profit[enabled]")

	// flip_operator_enabled
	var flipOperatorEnabled bool
	enabled := c.PostForm("entry[flip_operator_enabled]")
	switch enabled {
	case "1":
		flipOperatorEnabled = true
	case "0":
		flipOperatorEnabled = false
	default:
		return map[string]interface{}{}, errors.New("flip_operator_enabled is invalid")
	}

	// Prepare contract params
	contractParams := map[string]interface{}{
		"entry_type": c.PostForm("entry_type"),
		"entry_order": map[string]interface{}{
			"trigger": map[string]interface{}{
				"trigger_type": c.PostForm("entry[trigger_type]"),
				"operator":     c.PostForm("entry[operator]"),
				"price":        c.PostForm("entry[price]"),
			},
			"flip_operator_enabled": flipOperatorEnabled,
		},
	}
	if stopLossEnabled == "1" {
		contractParams["stop_loss_order"] = map[string]interface{}{
			"trigger": map[string]interface{}{
				"trigger_type": c.PostForm("stop_loss[trigger_type]"),
				"operator":     c.PostForm("stop_loss[operator]"),
				"price":        c.PostForm("stop_loss[price]"),
			},
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

	return contractParams, nil
}

func (ctl *Controller) processTrendlineContractParams(c *gin.Context) (map[string]interface{}, error) {
	params, err := ctl.convertTrendlineContractParams(c)
	if err != nil {
		return map[string]interface{}{}, err
	}

	// Stop-loss or take-profit enabled
	stopLossEnabled := c.PostForm("stop_loss[enabled]")
	takeProfitEnabled := c.PostForm("take_profit[enabled]")

	// Prepare contract params
	contractParams := map[string]interface{}{
		"entry_type": c.PostForm("entry_type"),
		"entry_order": map[string]interface{}{
			"trendline_trigger": map[string]interface{}{
				"trigger_type": c.PostForm("entry[trigger_type]"),
				"operator":     c.PostForm("entry[operator]"),
				"time_1":       params["time_1"].(time.Time).Format(time.RFC3339),
				"price_1":      c.PostForm("entry[price_1]"),
				"time_2":       params["time_2"].(time.Time).Format(time.RFC3339),
				"price_2":      c.PostForm("entry[price_2]"),
			},
			"trendline_offset_percent": params["trendline_offset_percent"].(float64),
			"flip_operator_enabled":    params["flip_operator_enabled"].(bool),
		},
	}
	if stopLossEnabled == "1" {
		contractParams["stop_loss_order"] = map[string]interface{}{
			"loss_tolerance_percent":         params["loss_tolerance_percent"].(float64),
			"trendline_readjustment_enabled": params["trendline_readjustment_enabled"].(bool),
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

	return contractParams, nil
}

func (ctl *Controller) convertTrendlineContractParams(c *gin.Context) (map[string]interface{}, error) {
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

	// trendline_offset_percent
	percent := c.PostForm("entry[trendline_offset_percent]")
	data["trendline_offset_percent"] = float64(0.0)
	if percent == "" {
		return data, errors.New("trendline_offset_percent is invalid")
	}
	p, err := strconv.ParseFloat(percent, 64)
	if err != nil {
		return data, errors.New("trendline_offset_percent is invalid")
	}
	data["trendline_offset_percent"] = float64(int64(p/100*10000)) / 10000 // convert to percent first, then fix float64

	// flip_operator_enabled
	data["flip_operator_enabled"] = false
	enabled := c.PostForm("entry[flip_operator_enabled]")
	switch enabled {
	case "1":
		data["flip_operator_enabled"] = true
	case "0":
		data["flip_operator_enabled"] = false
	default:
		return data, errors.New("flip_operator_enabled is invalid")
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

	// trendline_readjustment_enabled
	if stopLossEnabled == "1" {
		enabled = c.PostForm("stop_loss[trendline_readjustment_enabled]")
		data["trendline_readjustment_enabled"] = false
		if enabled == "" {
			return data, errors.New("trendline_readjustment_enabled is invalid")
		}
		if enabled != "1" && enabled != "0" {
			return data, errors.New("trendline_readjustment_enabled is invalid")
		}
		if enabled == "1" {
			data["trendline_readjustment_enabled"] = true
		}
		if enabled == "0" {
			data["trendline_readjustment_enabled"] = false
		}
	}

	return data, nil
}
