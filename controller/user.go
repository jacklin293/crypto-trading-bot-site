package controller

import (
	"crypto-trading-bot-engine/util/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func (ctl *Controller) NewApiKey(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}
	success := c.Query("success")
	errMsg := c.Query("err")

	c.HTML(http.StatusOK, "new_apikey.html", gin.H{
		"loggedIn": true,
		"success":  success,
		"errMsg":   errMsg,
	})
}

func (ctl *Controller) UpdateApiKey(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	apiKey := c.PostForm("api_key")
	apiSecret := c.PostForm("api_secret")
	subaccount := c.PostForm("subaccount")
	if apiKey == "" || apiSecret == "" || subaccount == "" {
		ctl.redirectToLoginPage(c, "/user/apikey/new?err=empty_data")
		return
	}

	details := make(map[string]interface{})
	details = map[string]interface{}{
		viper.GetString("DEFAULT_EXCHANGE"): map[string]interface{}{
			"api_key":    apiKey,
			"api_secret": apiSecret,
			"subaccount": subaccount,
		},
	}

	b, err := json.Marshal(details)
	if err != nil {
		log.Println("UpdateApiKey err:", err)
		ctl.redirectToLoginPage(c, "/user/apikey/new?err=internal_error")
		return
	}

	// Encrypt data using AES
	key, err := hex.DecodeString(viper.GetString("AES_PRIVATE_KEY"))
	if err != nil {
		log.Println("UpdateApiKey err:", err)
		ctl.redirectToLoginPage(c, "/user/apikey/new?err=internal_error")
		return
	}
	iv64, data64, err := aes.Encrypt([]byte(key), b)
	if err != nil {
		log.Println("UpdateApiKey err:", err)
		ctl.redirectToLoginPage(c, "/user/apikey/new?err=internal_error")
		return
	}
	encryptedData := fmt.Sprintf("%s;%s", iv64, data64)

	// Update data
	userCookie := ctl.getUserData(c)
	data := map[string]interface{}{
		"exchange_api_key": encryptedData,
	}
	if _, err = ctl.db.UpdateUser(userCookie.Uuid, data); err != nil {
		ctl.redirectToLoginPage(c, "/user/apikey/new?err=internal_error")
		return
	}

	ctl.redirectToLoginPage(c, "/user/apikey/new?success=update")
}

func (ctl *Controller) DeleteApiKey(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	userCookie := ctl.getUserData(c)
	data := map[string]interface{}{
		"exchange_api_key": nil,
	}
	if _, err := ctl.db.UpdateUser(userCookie.Uuid, data); err != nil {
		failJSONWithVagueError(c, "DeleteApiKey", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) TestApiKey(c *gin.Context) {
	if !ctl.tokenAuthCheck(c) {
		return
	}

	_, err := ctl.getExchangeAccountInfo(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "發生未知錯誤, 請更新 API Key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
