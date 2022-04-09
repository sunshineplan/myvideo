package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sunshineplan/utils/retry"
)

var (
	urlParse         = url.Parse
	urlQueryEscape   = url.QueryEscape
	urlQueryUnescape = url.QueryUnescape
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
	router.TrustedPlatform = "X-Real-IP"
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

	router.POST("/playlist", func(c *gin.Context) {
		url := c.PostForm("url")
		if url == "" {
			c.String(400, "")
			return
		}

		var playlist map[string][]play
		var err error
		if err := retry.Do(func() error {
			playlist, err = loadPlayList(url)
			return err
		}, 3, 3); err != nil {
			log.Print(err)
			c.String(500, "")
			return
		}

		c.JSON(200, playlist)
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
		if err := retry.Do(func() (err error) {
			url, err = loadPlay(play.URL, play.Play)
			return
		}, 3, 3); err != nil {
			log.Print(err)
			c.String(500, "")
			return
		}

		if testM3U8(url) {
			c.String(200, fmt.Sprint("/m3u8?ref=", urlQueryEscape(url)))
		} else {
			c.String(200, urlQueryEscape(url))
		}
	})

	router.GET("/m3u8", func(c *gin.Context) {
		url := c.Query("ref")
		url, err := urlQueryUnescape(url)
		if err != nil {
			log.Print(err)
			c.String(404, "")
			return
		}

		res, err := loadM3U8(url)
		if err == nil {
			c.Data(200, "application/vnd.apple.mpegurl", []byte(res))
		} else {
			log.Print(err)
			c.String(404, "")
		}
	})

	router.NoRoute(func(c *gin.Context) {
		c.Redirect(302, "/")
	})

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
