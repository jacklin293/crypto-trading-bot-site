package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"gorm.io/datatypes"
)

const (
	ENGINE_REQUEST_TIMEOUT_SECOND = 5
)

func (ctl *Controller) EnableStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}
	userCookie := ctl.getUserData(c)
	uuid := c.Param("uuid")

	// Check permission
	_, err := ctl.db.GetContractStrategyByUuidByUser(uuid, userCookie.Uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	// Send request to engine
	path := fmt.Sprintf("/event?action=enable&uuid=%s", uuid)
	_, err = ctl.makeRequestToEngine(path)
	if err != nil {
		ctl.log.Println("failed to call engine, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	// Update DB
	data := map[string]interface{}{
		"enabled": 1,
	}
	if _, err := ctl.db.UpdateContractStrategy(uuid, data); err != nil {
		ctl.log.Println("failed to update db, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) DisableStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}
	userCookie := ctl.getUserData(c)
	uuid := c.Param("uuid")

	// Check permission
	_, err := ctl.db.GetContractStrategyByUuidByUser(uuid, userCookie.Uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	// Send request to engine
	path := fmt.Sprintf("/event?action=disable&uuid=%s", uuid)
	_, err = ctl.makeRequestToEngine(path)
	if err != nil {
		// NOTE Allow strategy to be disabled while engine server is down
		ctl.log.Println("failed to call engine, err:", err)
	}

	// Update DB
	data := map[string]interface{}{
		"enabled": 0,
	}
	if _, err := ctl.db.UpdateContractStrategy(uuid, data); err != nil {
		ctl.log.Println("failed to update db, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) ResetStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}
	userCookie := ctl.getUserData(c)
	uuid := c.Param("uuid")

	// Check permission
	_, err := ctl.db.GetContractStrategyByUuidByUser(uuid, userCookie.Uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	// Make sure it's not tracked by engine
	if err = ctl.notBeingTrackedByEngine(c, uuid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update DB
	data := map[string]interface{}{
		"enabled":                 0,
		"position_status":         int64(contract.CLOSED),
		"exchange_orders_details": datatypes.JSONMap{},
	}
	if _, err := ctl.db.UpdateContractStrategy(uuid, data); err != nil {
		ctl.log.Println("failed to update db, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) ShareStrategy(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	// TODO check permission
	c.JSON(http.StatusOK, gin.H{"ShareStrategy": ""})
}

func (ctl *Controller) ClosePosition(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}
	userCookie := ctl.getUserData(c)
	uuid := c.Param("uuid")

	// Check permission
	cs, err := ctl.db.GetContractStrategyByUuidByUser(uuid, userCookie.Uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission denied"})
		return
	}

	// Check if the position is opened
	if contract.Status(cs.PositionStatus) != contract.OPENED {
		c.JSON(http.StatusBadRequest, gin.H{"error": "此策略並未開倉"})
		return
	}

	// Check if order details exist
	if len(cs.ExchangeOrdersDetails) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	// Make sure it's not tracked by engine
	if err = ctl.notBeingTrackedByEngine(c, uuid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Close position and stop-loss order
	if err := ctl.closePosition(c, cs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Unset some params
	params, err := ctl.unsetStopLossParamsAfterClosingPosition(cs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	// Update DB
	data := map[string]interface{}{
		"params":                  params,
		"enabled":                 0,
		"position_status":         int64(contract.CLOSED),
		"exchange_orders_details": datatypes.JSONMap{},
	}
	if _, err := ctl.db.UpdateContractStrategy(uuid, data); err != nil {
		ctl.log.Println("failed to update db, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) makeRequestToEngine(path string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * time.Duration(ENGINE_REQUEST_TIMEOUT_SECOND),
	}
	response, err := client.Get(viper.GetString("ENGINE_URL") + path)
	if err != nil {
		return []byte{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("status code: %d", response.StatusCode)
	}
	// NOTE for debug
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []byte{}, err
	}
	return body, err
}

func (ctl *Controller) notBeingTrackedByEngine(c *gin.Context, uuid string) error {
	path := fmt.Sprintf("/show?uuid=%s", uuid)
	resp, err := ctl.makeRequestToEngine(path)
	if err != nil {
		ctl.log.Println("failed to call engine, err:", err)
		return errors.New("Internal error")
	}
	body := map[string]interface{}{}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		ctl.log.Println("failed to unmarshal body from engine, err:", err)
		return errors.New("Internal error")
	}
	exist, ok := body["exist"].(bool)
	if !ok {
		ctl.log.Println("key 'exist' doesn't exist in the response of '/show'")
		return errors.New("Internal error")
	}
	if exist {
		return errors.New("請先暫停此策略")
	}
	return nil
}

func (ctl *Controller) closePosition(c *gin.Context, cs *db.ContractStrategy) error {
	ex, err := ctl.newExchange(c)
	if err != nil {
		return err
	}

	positionInfo, err := ex.RetryGetPosition(cs.Symbol, 30, 2)
	if err != nil {
		ctl.log.Println("[ERROR] failed to get position, err:", err)
		return fmt.Errorf("%s server error: '%s'", cs.Exchange, err.Error())
	}

	// Place order
	// If size is zero, it means that it might be closed already
	if positionInfo["size"].(string) == "0" {
		return fmt.Errorf("無法平倉, 請到 %s APP 確認並重置狀態", cs.Exchange)
	}
	size, err := decimal.NewFromString(positionInfo["size"].(string))
	if err != nil {
		return fmt.Errorf("請重試或到 %s APP 操作並重置狀態", cs.Exchange)
	}

	if err = ex.RetryClosePosition(cs.Symbol, order.Side(cs.Side), size, 30, 2); err != nil {
		ctl.log.Println("[ERROR] failed to close position, err: ", err)
		return fmt.Errorf("%s server error: '%s', 請重試或到 %s APP 操作並重置狀態", cs.Exchange, err.Error(), cs.Exchange)
	}

	// Close stop-loss order
	var stopLossOrderId int64
	stopLossDetail, ok := cs.ExchangeOrdersDetails["stop_loss_order"].(map[string]interface{})
	if ok {
		stopLossOrderId = int64(stopLossDetail["order_id"].(float64))
		err = ex.RetryCancelOpenTriggerOrder(stopLossOrderId, 20, 2)
		if err != nil {
			ctl.log.Println("[ERROR] failed to cancel stop-loss order, err: ", err)
			return fmt.Errorf("無法取消停損訂單, %s server error: '%s'", cs.Exchange, err.Error())
		}
	}

	return nil
}

func (ctl *Controller) unsetStopLossParamsAfterClosingPosition(cs *db.ContractStrategy) (params datatypes.JSONMap, err error) {
	contract, err := contract.NewContract(order.Side(cs.Side), cs.Params)
	if err != nil {
		return
	}

	params = datatypes.JSONMap{
		"entry_type":  contract.EntryType,
		"entry_order": contract.EntryOrder,
	}
	if contract.StopLossOrder != nil {
		// Unset stop-loss trigger as it will ben generated after entry triggered
		if contract.EntryType == order.ENTRY_TRENDLINE {
			contract.StopLossOrder.(*order.StopLoss).UnsetTrigger()
		}

		params["stop_loss_order"] = contract.StopLossOrder
	}
	if contract.TakeProfitOrder != nil {
		params["take_profit_order"] = contract.TakeProfitOrder
	}
	return
}
