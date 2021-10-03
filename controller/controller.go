package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/message"
	"log"
	"net/http"
	"time"

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

	// Session store
	store := sessions.NewCookieStore([]byte(viper.GetString("SESSION_SECRET")))

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

func (ctl *Controller) tokenAuthCheck(c *gin.Context) {
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		log.Println("TokenAuthMiddleware err:", err)
		ctl.redirectToLoginPage(c, "/login?err=internal_error")
		return
	}

	if expiryTs, ok := session.Values["expiry_ts"].(int64); ok {
		if time.Now().Unix() > expiryTs {
			ctl.redirectToLoginPage(c, "/login?err=session_expired")
			return
		}
	} else {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return
	}

	if _, ok := session.Values["uuid"].(string); !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return
	}
	if _, ok := session.Values["telegram_chat_id"].(int64); !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return
	}
	if _, ok := session.Values["username"].(string); !ok {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return
	}
}

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
		log.Println("TokenAuthMiddleware err:", err)
	}
	session.Options.MaxAge = -1
	err = session.Save(c.Request, c.Writer)
	if err != nil {
		log.Println("TokenAuthMiddleware err:", err)
	}
}

func (ctl *Controller) redirectToLoginPage(c *gin.Context, urlPath string) {
	c.Redirect(http.StatusFound, urlPath)
}
