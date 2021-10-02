package controller

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	// "gopkg.in/ezzarghili/recaptcha-go.v4"
)

const (
	RECAPTCHA_VERIFY_URL = "https://www.google.com/recaptcha/api/siteverify"
)

// Post params
type UserLogin struct {
	Username          string `form:"username"`
	Password          string `form:"password"`
	RecaptchaResponse string `form:"g-recaptcha-response"`
}

// Post params
type UserOTP struct {
	Username          string `form:"username"`
	RecaptchaResponse string `form:"g-recaptcha-response"`
}

func (ctl *Controller) LoginGET(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{
		"recaptchaSiteKey": viper.GetString("RECAPTCHA_SITE_KEY"),
	})
}

// NOTE intentionally sleep 2 seconds for security purpose
func (ctl *Controller) LoginPOST(c *gin.Context) {
	var u UserLogin
	if err := c.ShouldBind(&u); err != nil {
		time.Sleep(time.Second * 2)
		failJSON(c, "LoginPOST", err)
		return
	}
	valid, err := ctl.checkRecaptcha(u.RecaptchaResponse)
	if err != nil {
		time.Sleep(time.Second * 2)
		failJSON(c, "LoginPOST", err)
		return
	}
	if !valid {
		time.Sleep(time.Second * 2)
		failJSON(c, "LoginPOST", errors.New("recaptcha returned invalid access"))
		return
	}

	// Get user
	time.Sleep(time.Second * 2)
	err = ctl.db.Login(u.Username, u.Password)
	if err != nil {
		failJSONWithVagueError(c, "LoginPOST", err)
		return
	}

	// TODO
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// NOTE intentionally sleep 2 seconds for security purpose
func (ctl *Controller) OTP(c *gin.Context) {
	var u UserOTP
	if err := c.ShouldBind(&u); err != nil {
		time.Sleep(time.Second * 2)
		failJSONWithVagueError(c, "OTP", err)
		return
	}
	valid, err := ctl.checkRecaptcha(u.RecaptchaResponse)
	if err != nil {
		time.Sleep(time.Second * 2)
		failJSONWithVagueError(c, "OTP", err)
		return
	}
	if !valid {
		time.Sleep(time.Second * 2)
		failJSONWithVagueError(c, "OTP", errors.New("failed to pass recaptcha"))
		return
	}

	// Get user
	time.Sleep(time.Second * 2)
	_, err = ctl.db.GetUserByName(u.Username)
	if err != nil {
		failJSONWithVagueError(c, "OTP", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (ctl *Controller) LogoutPOST(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"result": "ok"})
}

func (ctl *Controller) checkRecaptcha(response string) (bool, error) {
	var googleCaptcha string = viper.GetString("RECAPTCHA_SECRET")
	req, err := http.NewRequest("POST", RECAPTCHA_VERIFY_URL, nil)
	q := req.URL.Query()
	q.Add("secret", googleCaptcha)
	q.Add("response", response)
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	var googleResponse map[string]interface{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(body, &googleResponse)
	if err != nil {
		return false, err
	}
	success := googleResponse["success"].(bool)
	return success, nil
}
