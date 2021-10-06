package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"gorm.io/datatypes"
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
		log.Println("failed to call engine, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	// Update DB
	data := map[string]interface{}{
		"enabled": 1,
	}
	if _, err := ctl.db.UpdateContractStrategy(uuid, data); err != nil {
		log.Println("failed to update db, err:", err)
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
		log.Println("failed to call engine, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
		return
	}

	// Update DB
	data := map[string]interface{}{
		"enabled": 0,
	}
	if _, err := ctl.db.UpdateContractStrategy(uuid, data); err != nil {
		log.Println("failed to update db, err:", err)
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
		log.Println("failed to update db, err:", err)
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
	orderInfo, err := ctl.closeOpenPositionAndStopLossOrder(c, cs)
	if err != nil {
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
		log.Println("failed to update db, err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Internal error"})
	}

	c.JSON(http.StatusOK, gin.H{
		"price": orderInfo["price"].(string),
		"fee":   fmt.Sprintf("%.1f", orderInfo["fee"].(float64)),
	})
}

func (ctl *Controller) makeRequestToEngine(path string) ([]byte, error) {
	response, err := http.Get(viper.GetString("ENGINE_URL") + path)
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
		log.Println("failed to call engine, err:", err)
		return errors.New("Internal error")
	}
	body := map[string]interface{}{}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		log.Println("failed to unmarshal body from engine, err:", err)
		return errors.New("Internal error")
	}
	exist, ok := body["exist"].(bool)
	if !ok {
		log.Println("key 'exist' doesn't exist in the response of '/show'")
		return errors.New("Internal error")
	}
	if exist {
		return errors.New("請先暫停此策略")
	}
	return nil
}

func (ctl *Controller) closeOpenPositionAndStopLossOrder(c *gin.Context, cs *db.ContractStrategy) (map[string]interface{}, error) {
	size, err := decimal.NewFromString(cs.ExchangeOrdersDetails["entry_order"].(map[string]interface{})["size"].(string))
	if err != nil {
		log.Println("[ERROR] failed to get entry_order.size, err:", err)
		return map[string]interface{}{}, errors.New("內部錯誤, 請重置狀態")
	}

	ex, err := ctl.newExchange(c)
	if err != nil {
		return map[string]interface{}{}, err
	}

	// Place order
	orderId, err := ex.RetryClosePosition(cs.Symbol, order.Side(cs.Side), size, 30, 2)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid reduce-only order") {
			log.Println("[ERROR] order could be closed by FTX already, err: ", err)
			return map[string]interface{}{}, fmt.Errorf("訂單可能已經關閉,請到 %s APP 確認,確認後請重置狀態", cs.Exchange)
		}
		log.Println("[ERROR] failed to close position, err: ", err)
		return map[string]interface{}{}, fmt.Errorf("%s server responded: '%s'", cs.Exchange, err.Error())
	}

	// Check position is created
	orderInfo, count, err := ex.RetryGetPosition(orderId, 30, 2)
	if err != nil {
		log.Println("[ERROR] failed to get position, err: ", err)
		return map[string]interface{}{}, fmt.Errorf("%s server error: '%s'", cs.Exchange, err.Error())
	}
	if count == 0 {
		log.Println("[ERROR] no position was found")
		return map[string]interface{}{}, errors.New("未知錯誤,請確認訂單是否正確關閉,並且重置狀態")
	}

	// Close stop-loss order
	var stopLossOrderId int64
	stopLossDetail, ok := cs.ExchangeOrdersDetails["stop_loss_order"].(map[string]interface{})
	if ok {
		stopLossOrderId = int64(stopLossDetail["order_id"].(float64))
		err = ex.RetryCancelOpenTriggerOrder(stopLossOrderId, 20, 2)
		if err != nil {
			log.Println("[ERROR] failed to cancel stop-loss order, err: ", err)
			return map[string]interface{}{}, fmt.Errorf("無法取消停損訂單, %s server error: '%s'", cs.Exchange, err.Error())
		}
	}

	return orderInfo, nil
}
