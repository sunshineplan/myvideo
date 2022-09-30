package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/chrome"
	"github.com/sunshineplan/gohttp"
)

var category = map[string]string{
	"dongman":   "/dongman/",
	"dianying":  "/dianying/",
	"dianshiju": "/dianshiju/",
	"zongyi":    "/zongyi/",
}

type video struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Image string `json:"image"`
}

type play struct {
	EP   string `json:"ep"`
	M3U8 string `json:"m3u8"`
}

func getPage(s string) (int, error) {
	re1 := regexp.MustCompile(`index_(\d+)\.html`)
	re2 := regexp.MustCompile(`/so/.*-.*-(\d*)-.*\.html`)
	re3 := regexp.MustCompile(`/.+/.*-.*-.*-(\d*)-.*\.html`)

	if re1.MatchString(s) {
		return strconv.Atoi(re1.FindStringSubmatch(s)[1])
	} else if re2.MatchString(s) {
		return strconv.Atoi(re2.FindStringSubmatch(s)[1])
	} else if re3.MatchString(s) {
		return strconv.Atoi(re3.FindStringSubmatch(s)[1])
	}

	return 0, fmt.Errorf("fail to get page: %s", s)
}

func getList(path string) (list []video, total int, err error) {
	resp := gohttp.Get(*api+path, nil)
	if resp.Error != nil {
		err = resp.Error
		return
	}

	doc := soup.HTMLParse(resp.String())
	lists := doc.FindStrict("div", "class", "lists-content")
	if lists.Error != nil {
		err = fmt.Errorf("fail to get list: %v", lists.Error)
		return
	}

	li := lists.FindAll("li")
	if len(li) == 0 {
		return
	}
	for _, i := range li {
		list = append(list, video{
			Name:  i.Find("h2").FullText(),
			URL:   *api + i.Find("a").Attrs()["href"],
			Image: i.Find("img").Attrs()["src"],
		})
	}

	pagination := doc.Find("div", "class", "pagination")
	pages := pagination.FindAll("a")
	if len(pages) == 0 {
		total = 1
	} else {
		var href string
		for _, i := range pages {
			if i.Text() == "尾页" {
				href = i.Attrs()["href"]
				break
			}
		}

		total, err = getPage(href)
	}

	return
}

func getPlayList(url string) (map[string][]play, error) {
	c := chrome.Headless(false)
	if _, _, err := c.WithTimeout(time.Duration(*timeout) * time.Second); err != nil {
		return nil, err
	}
	defer c.Close()

	if err := c.EnableFetch(func(ev *fetch.EventRequestPaused) bool {
		return (ev.ResourceType == network.ResourceTypeDocument ||
			ev.ResourceType == network.ResourceTypeScript) &&
			strings.Contains(ev.Request.URL, "and")
	}); err != nil {
		return nil, err
	}

	var dl, db string
	if err := chromedp.Run(
		c,
		chromedp.Navigate(url),
		chromedp.WaitVisible("div.bd"),
		chromedp.InnerHTML("div#slider>header>dl", &dl),
		chromedp.InnerHTML("div.bd", &db),
	); err != nil {
		return nil, err
	}

	var keys []string
	for _, i := range soup.HTMLParse(dl).FindAll("dt") {
		keys = append(keys, i.Text())
	}

	var pp [][]play
	for _, i := range soup.HTMLParse(db).FindAll("ul") {
		var p []play
		for _, a := range i.FindAll("a") {
			p = append(p, play{a.Text(), a.Attrs()["onclick"]})
		}
		pp = append(pp, p)
	}

	list := make(map[string][]play)
	for index, i := range keys {
		list[i] = pp[1 : len(pp)-1][index]
	}

	return list, nil
}

func getPlay(play, script string) (url string, err error) {
	c := chrome.Headless(false)
	if _, _, err = c.WithTimeout(time.Duration(*timeout) * time.Second); err != nil {
		return
	}
	defer c.Close()

	if err = c.EnableFetch(func(ev *fetch.EventRequestPaused) bool {
		return ((ev.ResourceType == network.ResourceTypeDocument ||
			ev.ResourceType == network.ResourceTypeScript ||
			ev.ResourceType == network.ResourceTypeXHR) &&
			regexp.MustCompile("and|npm|m3u8").MatchString(ev.Request.URL))
	}); err != nil {
		return
	}

	if err = chromedp.Run(c, chromedp.Navigate(play), chromedp.WaitVisible("div.bd")); err != nil {
		return
	}

	_, done, err := c.ListenScriptEvent(script, nil, "", "", true)
	if err != nil {
		return
	}

	select {
	case <-c.Done():
		return "", c.Err()
	case e := <-done:
		if url = e.Request.Request.URL; url != *api+"/url.php" {
			return
		}
		url = string(e.Bytes)
		_, err = urlParse(url)
		return
	}
}

func testM3U8(url string) bool {
	resp := gohttp.Head(url, nil)
	if resp.Error != nil {
		return true
	}
	if resp.StatusCode != 200 {
		return true
	}
	return resp.ContentLength < 3*1024*1024
}
