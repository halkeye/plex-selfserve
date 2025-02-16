package main

import (
	"context"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/openidConnect"
	authentik "goauthentik.io/api/v3"
)

type config struct {
	SessionKey        string `env:"SESSION_SECRET,required"`
	ClientID          string `env:"CLIENT_ID,required"`
	ClientSecret      string `env:"CLIENT_SECRET,required"`
	ClientCallbackURL string `env:"CLIENT_CALLBACK_URL,required"`
	ClientIssuerURL   string `env:"CLIENT_ISSUER_URL,required"`
}

var cfg config
var store *sessions.CookieStore

const sessionName = "auth"

func init() {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Panicf("Error loading .env file: %v", err)
	}
}

func main() {
	var err error

	// parse
	err = env.Parse(&cfg)
	if err != nil {
		log.Panic(err)
	}

	store = sessions.NewCookieStore([]byte(cfg.SessionKey))
	gothic.Store = store

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	provider, err := openidConnect.New(cfg.ClientID, cfg.ClientSecret, cfg.ClientCallbackURL, cfg.ClientIssuerURL)
	if err != nil {
		log.Panic(err)
	}
	goth.UseProviders(provider)

	r.GET("/", home)
	r.GET("/auth/:provider", signInWithProvider)
	r.GET("/auth/:provider/callback", callbackHandler)
	r.GET("/success", Success)

	r.Run(":3000")

}

func Success(c *gin.Context) {
	session, err := store.Get(c.Request, sessionName)
	if err != nil {
		log.Panic(err)
	}

	// user := session.Values["user"].(goth.User)
	// fmt.Print(user)

	authentikCfg := authentik.NewConfiguration()
	authentikCfg.Host = "auth.g4v.dev"
	authentikCfg.Scheme = "https"

	authentikClient := authentik.NewAPIClient(authentikCfg)
	authCtx := context.WithValue(context.Background(), authentik.ContextAccessToken, session.Values["accessToken"])
	apps, _, err := authentikClient.AdminApi.AdminAppsListExecute(authentikClient.AdminApi.AdminAppsList(authCtx))

	if err != nil {
		log.Panic(err)
	}

	log.Print(apps)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`
      <div style="
          background-color: #fff;
          padding: 40px;
          border-radius: 8px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
          text-align: center;
      ">
          <h1 style="
              color: #333;
              margin-bottom: 20px;
          ">You have Successfull signed in!</h1>
          
          </div>
      </div>
  `))
}

func callbackHandler(c *gin.Context) {
	provider := c.Param("provider")
	q := c.Request.URL.Query()
	q.Add("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	session, err := store.Get(c.Request, sessionName)
	if err != nil {
		log.Panic(err)
	}

	// session.Values["user"] = user
	session.Values["accessToken"] = user.AccessToken
	err = session.Save(c.Request, c.Writer)
	if err != nil {
		log.Panic("Problem Saving session data", err)
	}

	c.Redirect(http.StatusTemporaryRedirect, "/success")
}

func signInWithProvider(c *gin.Context) {
	provider := c.Param("provider")
	q := c.Request.URL.Query()
	q.Add("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func home(c *gin.Context) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(c.Writer, gin.H{})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

}
