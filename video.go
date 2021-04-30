package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/anaskhan96/soup"
	"github.com/robertkrimen/otto"
	"github.com/sunshineplan/gohttp"
	"github.com/sunshineplan/utils"
)

var category = map[string]string{
	"1": "/dongman/",
	"2": "/dianying/",
	"3": "/dianshiju/",
	"4": "/zongyi/",
}

var vm = otto.New()

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

func getList(arg string) ([]video, error) {
	resp := gohttp.Get(api+arg, nil)
	if resp.Error != nil {
		return nil, resp.Error
	}

	var videos []video
	for _, i := range soup.HTMLParse(resp.String()).FindStrict("div", "class", "lists-content").FindAll("li") {
		videos = append(videos, video{
			Name:  i.Find("h2").FullText(),
			URL:   api + i.Find("a").Attrs()["href"],
			Image: i.Find("img").Attrs()["src"],
		})
	}

	var wg sync.WaitGroup
	wg.Add(len(videos))
	for i := range videos {
		go func(v *video) {
			defer wg.Done()
			if err := utils.Retry(v.getPlayList, 3, 1); err != nil {
				log.Print(err)
			}
		}(&videos[i])
	}
	wg.Wait()

	return videos, nil
}

func (v *video) getPlayList() error {
	resp := gohttp.Get(v.URL, nil)
	if resp.Error != nil {
		return resp.Error
	}

	var script string
	for _, i := range soup.HTMLParse(resp.String()).FindAll("script") {
		if strings.Contains(i.Text(), "links") {
			script = i.Text()
			break
		}
	}

	if _, err := vm.Run(script); err != nil {
		return err
	}

	links, err := getVar("links")
	if err != nil {
		return err
	}

	var e encrypted
	if err := json.Unmarshal([]byte(links), &e); err != nil {
		return err
	}

	classid, err := getVar("classid")
	if err != nil {
		return err
	}
	infoid, err := getVar("infoid")
	if err != nil {
		return err
	}

	keyURL := fmt.Sprintf(api+"/e/extend/lgyPl2.0/?id=%v&classid=%v", infoid, classid)
	resp = gohttp.Get(keyURL, nil)
	if resp.Error != nil {
		return resp.Error
	}

	re := regexp.MustCompile(`e=(\d{5,6})`)
	key := re.FindStringSubmatch(resp.String())
	if key == nil {
		return fmt.Errorf("fail to get key from %s", keyURL)
	}

	e.Key = "dandan" + key[1]
	result, err := e.decrypt()
	if err != nil {
		return err
	}

	mu.Lock()
	v.PlayList = make(map[string][]play)
	mu.Unlock()

	for _, i := range strings.Split(result, "|@@@") {
		if i != "" {
			s := strings.Split(i, "!!!")
			if len(s) != 2 {
				return fmt.Errorf("strings split error: %s", i)
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
					return fmt.Errorf("strings split error: %s", ep)
				}
				eps = append(eps, play{EP: s[0], M3U8: s[1]})
			}

			mu.Lock()
			v.PlayList[key] = eps
			mu.Unlock()
		}
	}

	return nil
}

func getVar(key string) (string, error) {
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
