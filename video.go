package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/gohttp"
	"github.com/sunshineplan/utils"
	"github.com/sunshineplan/utils/workers"
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
var timeout = 20 * time.Second

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

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	workers.Slice(list, func(i int, _ interface{}) {
		if err := utils.Retry(func() error {
			return (&list[i]).getPlayList(ctx)
		}, 2, 3); err != nil {
			log.Print(err)
		}
	})

	return
}

func getPlayList(url string, ctx context.Context) (map[string][]play, error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				c := chromedp.FromContext(ctx)
				ctx := cdp.WithExecutor(ctx, c.Target)

				if (ev.ResourceType == network.ResourceTypeDocument ||
					ev.ResourceType == network.ResourceTypeScript) &&
					strings.Contains(ev.Request.URL, "and") {
					fetch.ContinueRequest(ev.RequestID).Do(ctx)
				} else {
					fetch.FailRequest(ev.RequestID, network.ErrorReasonBlockedByClient).Do(ctx)
				}
			}()
		}
	})

	var dl, db string
	if err := chromedp.Run(
		ctx,
		runtime.Disable(),
		fetch.Enable(),
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

func (v *video) getPlayList(ctx context.Context) error {
	mu.Lock()
	url := v.URL
	mu.Unlock()

	playList, err := loadPlayList(url, ctx)
	if err != nil {
		return err
	}

	mu.Lock()
	v.PlayList = playList
	mu.Unlock()

	return nil
}

func getPlay(play, script string) (url string, err error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				c := chromedp.FromContext(ctx)
				ctx := cdp.WithExecutor(ctx, c.Target)

				if ((ev.ResourceType == network.ResourceTypeDocument ||
					ev.ResourceType == network.ResourceTypeScript ||
					ev.ResourceType == network.ResourceTypeXHR) &&
					regexp.MustCompile("and|npm").MatchString(ev.Request.URL)) ||
					ev.Request.URL == url {
					fetch.ContinueRequest(ev.RequestID).Do(ctx)
				} else {
					fetch.FailRequest(ev.RequestID, network.ErrorReasonBlockedByClient).Do(ctx)
				}
			}()
		}
	})

	var id network.RequestID
	done := make(chan bool)
	if err = chromedp.Run(
		ctx,
		runtime.Disable(),
		fetch.Enable(),
		chromedp.Navigate(play),
		chromedp.WaitVisible("div.bd"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.ListenTarget(ctx, func(v interface{}) {
				switch ev := v.(type) {
				case *network.EventRequestWillBeSent:
					if url == "" {
						url = ev.Request.URL
						if url != *api+"/url.php" {
							close(done)
							return
						}
						id = ev.RequestID

					}
				case *network.EventLoadingFinished:
					if ev.RequestID == id {
						close(done)
					}
				}
			})
			return nil
		}),
		chromedp.Evaluate(script, nil),
	); err != nil {
		return
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-done:
	}

	if url != *api+"/url.php" {
		return
	}

	err = chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			body, err := network.GetResponseBody(id).Do(ctx)
			if err != nil {
				return err
			}
			url = string(body)

			return nil
		}),
	)

	return
}
