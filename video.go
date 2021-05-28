package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/anaskhan96/soup"
	"github.com/robertkrimen/otto"
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

func getVar(key string, vm *otto.Otto) (string, error) {
	value, err := vm.Get(key)
	if err != nil {
		return "", err
	}

	if value.IsUndefined() {
		return "", fmt.Errorf("undefined value of %s", key)
	}

	v, err := value.Export()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", v), nil
}

func getPlayList(url string) (map[string][]play, error) {
	resp := gohttp.Get(url, nil)
	if resp.Error != nil {
		return nil, resp.Error
	}

	var script string
	for _, i := range soup.HTMLParse(resp.String()).FindAll("script") {
		if strings.Contains(i.Text(), "links") {
			script = i.Text()
			break
		}
	}

	vm := otto.New()
	var s []string
	for _, i := range strings.Split(script, ";") {
		if _, err := vm.Run(i); err != nil {
			//log.Print(i)
			continue
		}
		s = append(s, i)
	}
	if _, err := vm.Run(strings.Join(s, ";")); err != nil {
		return nil, err
	}

	links, err := getVar("links", vm)
	if err != nil {
		return nil, err
	}

	var e encrypted
	if err := json.Unmarshal([]byte(links), &e); err != nil {
		return nil, err
	}

	classid, err := getVar("classid", vm)
	if err != nil {
		return nil, err
	}
	infoid, err := getVar("infoid", vm)
	if err != nil {
		return nil, err
	}

	keyURL := fmt.Sprintf(api+"/e/extend/lgyPl2.0/?id=%v&classid=%v", infoid, classid)
	resp = gohttp.Get(keyURL, nil)
	if resp.Error != nil {
		return nil, resp.Error
	}

	re := regexp.MustCompile(`e=(\d{5,6})`)
	key := re.FindStringSubmatch(resp.String())
	if key == nil {
		return nil, fmt.Errorf("fail to get key from %s", keyURL)
	}

	e.Key = "dandan" + key[1]
	result, err := e.decrypt()
	if err != nil {
		return nil, err
	}

	playList := make(map[string][]play)

	for _, i := range strings.Split(result, "|@@@") {
		if i != "" {
			s := strings.Split(i, "!!!")
			if len(s) != 2 {
				return nil, fmt.Errorf("strings split error: %s", i)
			}
			key := s[0]
			var eps []play
			for _, ep := range strings.Split(s[1], "|") {
				if ep == "暂无资源" {
					eps = append(eps, play{EP: "暂无资源"})
					continue
				}
				s := strings.Split(ep, "$")
				if len(s) < 2 {
					return nil, fmt.Errorf("strings split error: %s", ep)
				}
				eps = append(eps, play{EP: s[0], M3U8: s[1]})
			}

			playList[key] = eps
		}
	}

	return playList, nil
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
