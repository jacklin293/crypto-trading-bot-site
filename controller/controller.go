package controller

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/message"
	"encoding/hex"
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
	log    *log.Logger
}

type UserData struct {
	Uuid string
	Role int64
}

func InitController(l *log.Logger) *Controller {
	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
	if err != nil {
		l.Fatal(err)
	}

	// Sender
	data := map[string]interface{}{
		"token": viper.Get("TELEGRAM_TOKEN"),
	}
	sender, err := message.NewSender(viper.GetString("DEFAULT_SENDER_PLATFORM"), data)
	if err != nil {
		l.Fatal(err)
	}

	authKey, err := hex.DecodeString(viper.GetString("SESSION_AUTHENTICATION_KEY"))
	if err != nil {
		l.Fatal(err)
	}
	encryptKey, err := hex.DecodeString(viper.GetString("SESSION_ENCRYPTION_KEY"))
	if err != nil {
		l.Fatal(err)
	}

	// Session store
	store := sessions.NewCookieStore(authKey, encryptKey)

	return &Controller{
		db:     db,
		sender: sender,
		store:  store,
		log:    l,
	}
}

// NOTE intentionally provide vague for security purpose
func (ctl *Controller) failJSONWithVagueError(c *gin.Context, caller string, err error) {
	ctl.log.Printf("[ERROR] %s err: %s", caller, err.Error())
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "Internal error",
	})
}

// must be called after 'tokenAuthCheck'
func (ctl *Controller) getUserData(c *gin.Context) *UserData {
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		ctl.failJSONWithVagueError(c, "getUserData", err)
		return &UserData{}
	}
	return &UserData{
		Uuid: session.Values["uuid"].(string),
		Role: session.Values["role"].(int64),
	}
}

func (ctl *Controller) clearSession(c *gin.Context) {
	session, err := ctl.store.Get(c.Request, "user-session")
	if err != nil {
		ctl.log.Println("clearSession err:", err)
	}
	session.Options.MaxAge = -1
	if err = session.Save(c.Request, c.Writer); err != nil {
		ctl.log.Println("clearSession err:", err)
	}
}

func (ctl *Controller) redirectToLoginPage(c *gin.Context, urlPath string) {
	c.Redirect(http.StatusFound, urlPath)
	c.Abort()
}
