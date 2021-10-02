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
	user   map[string]interface{}
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
		"success": false,
		"error":   "Internal error",
	})
}

func (ctl *Controller) tokenAuthCheck(c *gin.Context) {
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		log.Println("TokenAuthMiddleware err:", err)
		ctl.redirectToLoginPage(c, "/login?err=internal_error")
		return
	}

	expiryTs, ok := session.Values["expiry_ts"].(int64)
	if ok {
		if time.Now().Unix() > expiryTs {
			ctl.redirectToLoginPage(c, "/login?err=session_expired")
			return
		}
	} else {
		ctl.redirectToLoginPage(c, "/login?err=please_login")
		return
	}

	uuid := session.Values["uuid"].(string)
	chatId := session.Values["telegram_chat_id"].(int64)
	username := session.Values["username"].(string)

	ctl.user = map[string]interface{}{
		"uuid":             uuid,
		"telegram_chat_id": chatId,
		"username":         username,
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
