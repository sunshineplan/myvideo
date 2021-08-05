package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sunshineplan/utils"
)

func test() error {
	list, total, err := loadList("/dongman/")
	if err != nil {
		return err
	}
	if l := len(list); l == 0 || total == 0 {
		return fmt.Errorf("not expected result. length: %d, total: %d", l, total)
	}

	return nil
}

func run() {
	if *logPath != "" {
		f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			log.Fatalln("Failed to open log file:", err)
		}
		gin.DefaultWriter = f
		gin.DefaultErrorWriter = f
		log.SetOutput(f)
	}

	router := gin.Default()
	server.Handler = router

	router.StaticFS("/build", http.Dir(filepath.Join(filepath.Dir(self), "public/build")))
	router.StaticFS("/res", http.Dir(filepath.Join(filepath.Dir(self), "public/res")))
	router.StaticFile("favicon.ico", filepath.Join(filepath.Dir(self), "public/favicon.ico"))
	router.LoadHTMLFiles(
		filepath.Join(filepath.Dir(self), "public/index.html"),
		filepath.Join(filepath.Dir(self), "public/player.html"),
	)

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.POST("/list", func(c *gin.Context) {
		var data filter
		if err := c.BindJSON(&data); err != nil {
			log.Print(err)
			c.String(400, "")
			return
		}
		if !data.verify() {
			log.Println("unknow format:", data.string())
			c.String(400, "")
			return
		}

		var path string
		if data.Search == "" {
			var ok bool
			path, ok = category[c.Query("c")]
			if !ok {
				path = category["dongman"]
			}
		}
		path += data.string()

		list, total, err := loadList(path)
		if err != nil {
			log.Print(err)
			c.String(500, "")
			return
		}

		c.JSON(200, gin.H{"total": total, "list": list})
	})

	router.GET("/play", func(c *gin.Context) {
		c.HTML(200, "player.html", nil)
	})

	router.POST("/play", func(c *gin.Context) {
		var play struct {
			URL, Play string
		}
		if err := c.BindJSON(&play); err != nil {
			c.String(400, "")
			return
		}

		var url string
		var err error
		if err := utils.Retry(func() error {
			url, err = loadPlay(play.URL, play.Play)
			return err
		}, 2, 3); err != nil {
			log.Print(err)
			c.String(500, "")
			return
		}

		c.String(200, url)
	})

	router.NoRoute(func(c *gin.Context) {
		c.Redirect(302, "/")
	})

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
