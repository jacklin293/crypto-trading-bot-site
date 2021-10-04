package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/message"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
)

type Controller struct {
	db     *db.DB
	sender message.Messenger
	store  *sessions.CookieStore
}

type UserData struct {
	Uuid           string
	TelegramChatId int64
	Username       string
}

func InitController() *Controller {
	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	// Sender
	data := map[string]interface{}{
		"token": viper.Get("TELEGRAM_TOKEN"),
	}
	sender, err := message.NewSender(viper.GetString("DEFAULT_SENDER_PLATFORM"), data)
	if err != nil {
		log.Fatal(err)
	}

	// base64 decode key
	authKey, err := base64.StdEncoding.DecodeString(viper.GetString("SESSION_AUTHENTICATION_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	encryptKey, err := base64.StdEncoding.DecodeString(viper.GetString("SESSION_ENCRYPTION_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	// Session store
	store := sessions.NewCookieStore(authKey, encryptKey)

	return &Controller{
		db:     db,
		sender: sender,
		store:  store,
	}
}

// NOTE intentionally provide vague for security purpose
func failJSONWithVagueError(c *gin.Context, caller string, err error) {
	log.Printf("[ERROR] %s err: %s", caller, err.Error())
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "Internal error",
	})
}

// must be called after 'tokenAuthCheck'
func (ctl *Controller) getUserData(c *gin.Context) *UserData {
	session, _ := ctl.store.Get(c.Request, "user-session")
	return &UserData{
		Uuid:           session.Values["uuid"].(string),
		TelegramChatId: session.Values["telegram_chat_id"].(int64),
		Username:       session.Values["username"].(string),
	}
}

func (ctl *Controller) clearSession(c *gin.Context) {
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		log.Println("clearSession err:", err)
	}
	session.Options.MaxAge = -1
	err = session.Save(c.Request, c.Writer)
	if err != nil {
		log.Println("clearSession err:", err)
	}
}

func (ctl *Controller) redirectToLoginPage(c *gin.Context, urlPath string) {
	c.Redirect(http.StatusFound, urlPath)
	c.Abort()
}
