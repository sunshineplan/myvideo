package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func run() {
	router := gin.Default()
	server.Handler = router

	if *logPath != "" {
		f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			log.Fatalln("Failed to open log file:", err)
		}
		gin.DefaultWriter = f
		gin.DefaultErrorWriter = f
		log.SetOutput(f)
	}

	router.StaticFS("/build", http.Dir(filepath.Join(filepath.Dir(self), "public/build")))
	router.StaticFS("/res", http.Dir(filepath.Join(filepath.Dir(self), "public/res")))
	router.StaticFile("favicon.ico", filepath.Join(filepath.Dir(self), "public/favicon.ico"))
	router.LoadHTMLFiles(filepath.Join(filepath.Dir(self), "public/index.html"))

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.GET("/list", func(c *gin.Context) {
		cat := c.Query("c")
		q := c.Query("q")

		var action string
		if q != "" {
			action = fmt.Sprintf("/so/-%s--onclick.html", q)
		} else {
			var ok bool
			action, ok = category[cat]
			if !ok {
				action = category["1"]
			}
		}

		list, err := loadList(action)
		if err != nil {
			log.Print(err)
			c.String(500, "")
			return
		}

		c.JSON(200, list)
	})

	router.NoRoute(func(c *gin.Context) {
		c.Redirect(302, "/")
	})

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
