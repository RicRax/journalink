package main

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/RicRax/journaLink/auth"
	"github.com/RicRax/journaLink/model"
	"github.com/RicRax/journaLink/routes"
)

var (
	store                  = cookie.NewStore([]byte("secret"))
	sd    auth.SessionData = auth.SessionData{}
)

func main() {
	r := setupRouter()

	auth.InitRand()
	sd.Init()

	r.Run(":8080")
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(sessions.Sessions("mysession", store))

	r.LoadHTMLGlob("front/*")

	db, err := gorm.Open(sqlite.Open("mydatabase.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(&model.Diary{}, &model.DiaryAccess{}, &model.User{})

	// login routes
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.POST("/login/authentication", func(c *gin.Context) {
		s := sessions.Default(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "cookie set error")
		}

		u, err := model.AuthenticateUser(db, c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, "could not authenticate")
			return
		}

		r := auth.RandSeq(10)
		sd.AuthState[r] = u.UID
		s.Set("token", r)
		s.Save()

		c.JSON(http.StatusOK, gin.H{"redirectPath": "/home"})
		return
	})

	// register route
	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})

	// OAuth2 routes
	r.GET("/oauth/redirect", func(c *gin.Context) {
		auth.HandleOAuth(db, c)
	})

	// home route after authentication
	r.GET("/home", func(c *gin.Context) {
		s := sessions.Default(c)
		t := s.Get("token")
		id, ok := sd.AuthState[t]

		if ok {
			routes.RenderHome(db, c, id)
		}
	})

	r.GET("/home/addDiary", func(c *gin.Context) {
		routes.RenderAddDiary(db, c)
	})

	// diary endpoints
	r.POST("/diary", func(c *gin.Context) {
		s := sessions.Default(c)
		t := s.Get("token")
		id, ok := sd.AuthState[t]

		if !ok {
			c.JSON(http.StatusInternalServerError, "could not identify token")
		}
		var info model.DiaryInfo
		if err := c.ShouldBindJSON(&info); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			fmt.Println(err)
			return
		}

		info.OwnerID = int(id)

		if info.DiaryID != 0 {
			model.UpdateDiary(db, info, c)
		} else {
			model.AddDiary(db, info, c)
		}
	})

	r.GET("/diary/:id", func(c *gin.Context) {
		model.GetDiary(db, c)
	})

	// r.GET("/diary/shared", func(c *gin.Context) {
	// 	getAllSharedDiaries(db, c)
	// })

	r.DELETE("/diary/:id", func(c *gin.Context) {
		model.DeleteDiary(db, c)
	})

	// user endpoints
	r.POST("/user", func(c *gin.Context) {
		model.AddUser(db, c)
	})

	r.GET("/user", func(c *gin.Context) {
		model.GetUser(db, c)
	})

	return r
}
