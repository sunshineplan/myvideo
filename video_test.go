package main

import (
	"strings"
	"testing"
)

func TestGetList(t *testing.T) {
	list, n, err := getList("/dongman/")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Errorf("expected non-empty list; got empty list")
	}
	if n <= 0 {
		t.Errorf("expected greater than zero; got %d", n)
	}
}

func TestSearch(t *testing.T) {
	list, n, err := getList("/so/go-go--.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Errorf("expected non-empty list; got empty list")
	}
	if n <= 0 {
		t.Errorf("expected greater than zero; got %d", n)
	}
}

func TestGetPlayList(t *testing.T) {
	list, err := getPlayList(*api + "/dongman/57677.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Errorf("expected non-empty list; got empty list")
	}
}

func TestGetPlay(t *testing.T) {
	m3u8, err := getPlay(*api+"/dongman/57677.html", "play(0,0);")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m3u8, "m3u8") {
		t.Errorf("not contains m3u8, got %s", m3u8)
	}
}
