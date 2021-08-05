package main

import (
	"context"
	"time"

	"github.com/sunshineplan/utils"
	"github.com/sunshineplan/utils/cache"
)

type result struct {
	list  []video
	total int
}

var c = cache.New(true)

func loadList(path string) (list []video, total int, err error) {
	value, ok := c.Get(path)
	if ok {
		list = value.(result).list
		total = value.(result).total
		return
	}

	if err = utils.Retry(func() error {
		list, total, err = getList(path)
		return err
	}, 2, 3); err != nil {
		return
	}

	c.Set(path, result{list: list, total: total}, 10*time.Minute, nil)

	return
}

func loadPlayList(url string, ctx context.Context) (playlist map[string][]play, err error) {
	value, ok := c.Get(url)
	if ok {
		playlist = value.(map[string][]play)
		return
	}

	playlist, err = getPlayList(url, ctx)
	if err != nil {
		return
	}

	c.Set(url, playlist, time.Hour, nil)

	return
}

func loadPlay(play, script string) (url string, err error) {
	value, ok := c.Get(play + script)
	if ok {
		url = value.(string)
		return
	}

	url, err = getPlay(play, script)
	if err != nil {
		return
	}

	c.Set(play+script, url, time.Hour, nil)

	return
}
