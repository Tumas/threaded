package threaded

import (
	"encoding/json"
	"fmt"
	rss "github.com/jteeuwen/go-pkg-rss"
	"io/ioutil"
	"reflect"
	"testing"
)

func readFixtureData(t *testing.T, path string) (*rss.Feed, []byte, chan *FeedResultBundle) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error("Unable to load fixture data")
	}

	feedItem := &FeedConfigItem{
		Guid: "geras_dviratis",
		Url:  "test",
		Identifier: &FeedIdentifier{
			ParamName: "t",
			ParamType: "parameter",
		},
	}

	feedChannel := make(chan *FeedResultBundle)
	itemHandler := spawnItemHandler(feedChannel, feedItem, 1)
	feed := rss.New(1, true, nil, itemHandler)

	return feed, content, feedChannel
}

func TestFetchingItems(t *testing.T) {
	feed, content, ch := readFixtureData(t, "fixtures/syndication")
	go feed.FetchBytes("http://example.com", content, nil)
	results := <-ch

	if len(results.Items) != 9 {
		t.Error("should have fetched 9 items")
	}
}

func TestFetchingSameResults(t *testing.T) {
	feed, content, ch := readFixtureData(t, "fixtures/syndication")
	go feed.FetchBytes("http://example.com", content, nil)
	go feed.FetchBytes("http://example.com", content, nil)

	<-ch
	select {
	case <-ch:
		t.Error("Should not receive items if nothing has changed")
	default:
	}
}

func TestThreadIDCollecting(t *testing.T) {
	feed, content, ch := readFixtureData(t, "fixtures/syndication")
	go feed.FetchBytes("http://example.com", content, nil)
	results := <-ch

	correct_keys := []string{"47649", "2968", "47531", "47524", "47677", "47613", "47325", "46951", "47669"}
	for _, k := range correct_keys {
		if _, ok := results.Items[k]; !ok {
			t.Error("Should have collected correct keys - missing: %s", k)
		}
	}
}

func TestThreadNestingKeyLength(t *testing.T) {
	feed, content, ch := readFixtureData(t, "fixtures/syndication_nested")
	go feed.FetchBytes("http://example.com", content, nil)
	results := <-ch

	if len(results.Items) != 8 {
		t.Error("should have fetched 8 items")
	}
}

func TestThreadNesting(t *testing.T) {
	feed, content, ch := readFixtureData(t, "fixtures/syndication_nested")
	go feed.FetchBytes("http://example.com", content, nil)
	results := <-ch

	if results.Items["47677"].MessageCount != 3 {
		t.Error("should have combined nested news under one thread")
	}
}

func TestJSONRepresentationOfResults(t *testing.T) {
	feed, content, ch := readFixtureData(t, "fixtures/syndication")
	go feed.FetchBytes("http://example.com", content, nil)
	results := <-ch

	b, err := json.Marshal(results.Items["47524"])
	if err != nil {
		t.Error(err)
	}

	res := make(map[string]interface{})
	in := map[string]interface{}{
		"title":           "Re: DEMA Quark XC FS remas",
		"last_updated_at": "Mon, 19 Aug 2013 21:05:52 +0300",
		"message_count":   1.0,
	}

	if err := json.Unmarshal(b, &res); err != nil {
		t.FailNow()
	}

	if !reflect.DeepEqual(in, res) {
		fmt.Printf("Expected %#v\n got %#v", in, res)
		t.FailNow()
	}
}
