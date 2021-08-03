package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/gohttp"
	"github.com/sunshineplan/utils"
)

var category = map[string]string{
	"dongman":   "/dongman/",
	"dianying":  "/dianying/",
	"dianshiju": "/dianshiju/",
	"zongyi":    "/zongyi/",
}

type video struct {
	Name     string            `json:"name"`
	URL      string            `json:"url"`
	Image    string            `json:"image"`
	PlayList map[string][]play `json:"playlist"`
}

type play struct {
	EP   string `json:"ep"`
	M3U8 string `json:"m3u8"`
}

var mu sync.Mutex

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
	resp := gohttp.Get(api+path, nil)
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
			URL:   api + i.Find("a").Attrs()["href"],
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

	var wg sync.WaitGroup
	wg.Add(len(list))
	for i := range list {
		go func(v *video) {
			defer wg.Done()
			if err := utils.Retry(v.getPlayList, 3, 1); err != nil {
				log.Print(err)
			}
		}(&list[i])
	}
	wg.Wait()

	return
}

func getPlayList(url string) (map[string][]play, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dl, db string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`div.bd`),
		chromedp.InnerHTML("div#slider>header>dl", &dl),
		chromedp.InnerHTML(`div.bd`, &db),
		chromedp.Stop(),
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

func (v *video) getPlayList() error {
	mu.Lock()
	url := v.URL
	mu.Unlock()

	playList, err := loadPlayList(url)
	if err != nil {
		return err
	}

	mu.Lock()
	v.PlayList = playList
	mu.Unlock()

	return nil
}
