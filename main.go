package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/openidConnect"
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

func main() {
	// parse
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	store = sessions.NewCookieStore([]byte(cfg.SessionKey))

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.ClientCallbackURL == "" {
		log.Fatal("Environment variables (CLIENT_ID, CLIENT_SECRET, CLIENT_CALLBACK_URL) are required")
	}

	fmt.Printf("Config: %+v\n", cfg)
	provider, err := openidConnect.New(cfg.ClientID, cfg.ClientSecret, cfg.ClientCallbackURL, cfg.ClientIssuerURL)
	if err != nil {
		log.Fatal(err)
	}
	goth.UseProviders(provider)

	r.GET("/", home)
	r.GET("/auth/:provider", signInWithProvider)
	r.GET("/auth/:provider/callback", callbackHandler)
	r.GET("/success", Success)

	r.Run(":3000")

}

func Success(c *gin.Context) {

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

	_, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
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
