package main

import (
	rss "github.com/jteeuwen/go-pkg-rss"
	"io/ioutil"
	"testing"
)

func readFixtureData(t *testing.T) (*rss.Feed, []byte, chan *FeedResultBundle) {
	content, err := ioutil.ReadFile("fixtures/syndication.php")
	if err != nil {
		t.Error("Unable to load fixture data")
	}

	feedItem := &FeedConfigItem{
		guid: "geras_dviratis",
		url:  "http://www.gerasdviratis.lt/forum/syndication.php",
		identifier: &FeedIdentifier{
			paramName: "t",
			paramType: "parameter",
		},
	}

	feedChannel := make(chan *FeedResultBundle)
	itemHandler := spawnItemHandler(feedChannel, feedItem, 1)
	feed := rss.New(1, true, nil, itemHandler)

	return feed, content, feedChannel
}

func TestFetchingItems(t *testing.T) {
	feed, content, ch := readFixtureData(t)
	go feed.FetchBytes("http://example.com", content, nil)
	results := <-ch

	if len(results.items) != 9 {
		t.Error("should have fetched 9 items")
	}
}

func TestFetchingSameResults(t *testing.T) {
	feed, content, ch := readFixtureData(t)
	go feed.FetchBytes("http://example.com", content, nil)
	go feed.FetchBytes("http://example.com", content, nil)

	<-ch
	select {
	case <-ch:
		t.Error("Should not receive items if nothing has changed")
	default:
	}
}
