package controller

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
)

const (
	RECAPTCHA_VERIFY_URL = "https://www.google.com/recaptcha/api/siteverify"

	// NOTE intentionally sleep a few seconds for security purpose
	RESPONSE_DELAY_SECOND = 1

	// The period for one-time password
	OTP_EXPIRY_SECOND = 180

	// The period for successfully login session
	LOGIN_EXPIRY_LENGTH_DAY = 7
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

func (ctl *Controller) LoginPage(c *gin.Context) {
	errType := c.Query("err")
	success := c.Query("success")
	c.HTML(200, "login.html", gin.H{
		"recaptchaSiteKey": viper.GetString("RECAPTCHA_SITE_KEY"),
		"otpExpirySecond":  OTP_EXPIRY_SECOND,
		"errType":          errType,
		"success":          success,
	})
}

func (ctl *Controller) LoginAPI(c *gin.Context) {
	time.Sleep(time.Second * time.Duration(RESPONSE_DELAY_SECOND))

	var u UserLogin
	if err := c.ShouldBind(&u); err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}
	valid, err := ctl.checkRecaptcha(u.RecaptchaResponse)
	if err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}
	if !valid {
		log.Println("LoginAPI err: invalid")
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}

	// Get user by username namd password
	user, err := ctl.db.GetUserByUsernameAndPassword(u.Username, u.Password)
	if err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}

	// Update user data
	data := map[string]interface{}{
		"last_login_at": time.Now(),
	}
	if _, err = ctl.db.UpdateUser(user.Uuid, data); err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}

	// Get session
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.clearSession(c)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}

	// Generate signature
	expiryTs := time.Now().Add(time.Second * 86400 * LOGIN_EXPIRY_LENGTH_DAY).Unix()
	hash, err := ctl.getSignatureHash(user.Uuid, user.Username, expiryTs)
	if err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}
	signature := base64.StdEncoding.EncodeToString(hash)

	session.Values["uuid"] = user.Uuid
	session.Values["telegram_chat_id"] = user.TelegramChatId
	session.Values["username"] = user.Username
	session.Values["expiry_ts"] = expiryTs
	session.Values["signature"] = signature
	session.Options = &sessions.Options{
		MaxAge: 86400 * LOGIN_EXPIRY_LENGTH_DAY,
	}
	err = session.Save(c.Request, c.Writer)
	if err != nil {
		log.Println("LoginAPI err: ", err)
		ctl.redirectToLoginPage(c, "/login?err=login_failed")
		return
	}

	ctl.redirectToLoginPage(c, "/?success=login")
}

// NOTE intentionally sleep 2 seconds for security purpose
func (ctl *Controller) OTP(c *gin.Context) {
	time.Sleep(time.Second * time.Duration(RESPONSE_DELAY_SECOND))

	var u UserOTP
	if err := c.ShouldBind(&u); err != nil {
		failJSONWithVagueError(c, "OTP", err)
		return
	}
	valid, err := ctl.checkRecaptcha(u.RecaptchaResponse)
	if err != nil {
		failJSONWithVagueError(c, "OTP", err)
		return
	}
	if !valid {
		failJSONWithVagueError(c, "OTP", errors.New("failed to pass recaptcha"))
		return
	}

	// Get user
	user, err := ctl.db.GetUserByUsername(u.Username)
	if err != nil {
		failJSONWithVagueError(c, "OTP", err)
		return
	}

	// Generate one-time password
	otp := uuid.New().String()

	// Send password via telegram
	ctl.sender.Send(user.TelegramChatId, "One-time password:")
	ctl.sender.Send(user.TelegramChatId, otp)

	// Update user
	data := map[string]interface{}{
		"password":            otp,
		"password_expired_at": time.Now().Add(time.Second * time.Duration(OTP_EXPIRY_SECOND)),
	}
	if _, err = ctl.db.UpdateUser(user.Uuid, data); err != nil {
		failJSONWithVagueError(c, "LoginPOST", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (ctl *Controller) Logout(c *gin.Context) {
	ctl.clearSession(c)
	ctl.redirectToLoginPage(c, "/login?success=true")
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

func (ctl *Controller) tokenAuthCheck(c *gin.Context) bool {
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		log.Println("tokenAuthCheck err:", err)
		ctl.redirectToLoginPage(c, "/login?err=internal_error")
		return false
	}

	expiryTs, ok := session.Values["expiry_ts"].(int64)
	if ok {
		if time.Now().Unix() > expiryTs {
			ctl.redirectToLoginPage(c, "/login?err=session_expired")
			return false
		}
	} else {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return false
	}

	if _, ok := session.Values["telegram_chat_id"].(int64); !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return false
	}
	uuid, ok := session.Values["uuid"].(string)
	if !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return false
	}
	username, ok := session.Values["username"].(string)
	if !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return false
	}

	// signature signed by server
	signature, ok := session.Values["signature"].(string)
	if !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return false
	}
	signatureHash, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		ctl.redirectToLoginPage(c, "/login?err=internal_error")
		return false
	}

	// hash gnenrated by cookie data
	hash, err := ctl.getSignatureHash(uuid, username, expiryTs)
	if err != nil {
		ctl.redirectToLoginPage(c, "/login?err=internal_error")
		return false
	}

	if !reflect.DeepEqual(signatureHash, hash) {
		ctl.redirectToLoginPage(c, "/login?err=internal_error")
		return false
	}

	return true
}

// signature = []byte({uuid}-{username}-{expiryTs}) + []byte(salt)
func (ctl *Controller) getSignatureHash(uuid string, username string, expiryTs int64) ([]byte, error) {
	// Get signature key
	salt, err := base64.StdEncoding.DecodeString(viper.GetString("SESSION_SIGNATURE_SALT"))
	if err != nil {
		return []byte{}, err
	}
	s := fmt.Sprintf("%s-%s-%d", uuid, username, expiryTs)
	sigKey := []byte(s)
	sigKey = append(sigKey, salt...)

	// Hash signature key
	h := sha256.New()
	_, err = h.Write([]byte(sigKey))
	if err != nil {
		return []byte{}, err
	}
	return h.Sum(nil), nil

}
