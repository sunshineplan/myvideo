package main

import (
	"fmt"
	"regexp"
)

type filter struct {
	Search string
	Genre  string
	Area   string
	Year   string
	Page   uint
	Sort   string
}

func (f *filter) verify() bool {
	if f.Genre != "" {
		if ok := check(
			f.Genre,
			[]string{"剧情", "喜剧", "动作", "爱情", "科幻", "悬疑", "惊悚", "恐怖", "犯罪", "同性",
				"音乐", "歌舞", "传记", "历史", "战争", "西部", "奇幻", "冒险", "灾难", "武侠", "伦理"},
		); !ok {
			return false
		}
	}

	if f.Area != "" {
		if ok := check(
			f.Area,
			[]string{"中国大陆", "美国", "香港", "台湾", "日本", "韩国", "英国", "法国", "德国", "意大利",
				"西班牙", "印度", "泰国", "俄罗斯", "伊朗", "加拿大", "澳大利亚", "爱尔兰", "瑞典", "巴西", "丹麦"},
		); !ok {
			return false
		}
	}

	if f.Year != "" {
		if ok := regexp.MustCompile(`\d+|\d+__\d+`).MatchString(f.Year); !ok {
			return false
		}
	}

	if f.Sort != "" {
		if ok := check(f.Sort, []string{"newstime", "onclick", "rating"}); !ok {
			return false
		}
	}

	return true
}

func (f *filter) string() string {
	if f.Search == "" {
		return fmt.Sprintf("%s-%s-%s-%d-%s.html", f.Genre, f.Area, f.Year, f.Page, f.Sort)
	}
	return fmt.Sprintf("/so/%s-%s-%d-%s.html", f.Search, f.Search, f.Page, f.Sort)
}

func check(s string, a []string) (ok bool) {
	for _, i := range a {
		if s == i {
			ok = true
		}
	}
	return
}
