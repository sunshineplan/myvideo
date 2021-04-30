package main

import (
	"time"

	"github.com/sunshineplan/utils"
	"github.com/sunshineplan/utils/cache"
)

var listCache = cache.New(false)

func loadList(action string) ([]video, error) {
	value, ok := listCache.Get(action)
	if ok {
		return value.([]video), nil
	}

	var list []video
	var err error
	if err := utils.Retry(func() error {
		list, err = getList(action)
		return err
	}, 3, 2); err != nil {
		return nil, err
	}

	listCache.Set(action, list, 10*time.Minute, nil)

	return list, nil
}
