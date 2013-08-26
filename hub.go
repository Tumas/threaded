package threaded

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	rss "github.com/jteeuwen/go-pkg-rss"
	"net/url"
	"os"
	"time"
)

// "github.com/coopernurse/gorp"

type FeedIdentifier struct {
	ParamName string
	ParamType string
}

type FeedConfigItem struct {
	Guid       string
	Url        string
	Identifier *FeedIdentifier
}

type FeedResult struct {
	Title         string
	LastUpdatedAt string
	MessageCount  int
}

type FeedResultBundle struct {
	Feed  *FeedConfigItem
	Items map[string]*FeedResult
}

type Hub struct {
	Connections map[*websocket.Conn]bool
	Register    chan *websocket.Conn
	Unregister  chan *websocket.Conn
}

func spawnItemHandler(mainChannel chan *FeedResultBundle, feedConfigItem *FeedConfigItem, timeout int) func(*rss.Feed, *rss.Channel, []*rss.Item) {
	previous := make(map[string]bool)
	current := make(map[string]bool)
	results := make(map[string]*FeedResult)

	return func(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
		ident := feedConfigItem.Identifier

		for _, item := range newitems {
			u, err := url.Parse(item.Guid)
			if err != nil {
				fmt.Printf("Error when parsing guid: %s\n", item.Guid)
				continue
			}

			if ident.ParamType == "parameter" {
				queryParams, err := url.ParseQuery(u.RawQuery)
				if err != nil {
					fmt.Printf("Error when parsing query params: %s\n", u.RawQuery)
					continue
				}

				branch := queryParams[ident.ParamName][0]
				// if branch, ok := queryParams[ident.paramName][0]; !ok {
				//  fmt.Printf("Could not find topic id! Config mismatch!")
				//  continue
				// }

				current[item.Guid] = true

				if _, ok := previous[item.Guid]; !ok {
					if resultRecord, ok := results[branch]; ok {
						resultRecord.LastUpdatedAt = item.PubDate //append(array, item)
						resultRecord.MessageCount += 1
					} else {
						results[branch] = &FeedResult{
							MessageCount:  1,
							Title:         item.Title,
							LastUpdatedAt: item.PubDate,
						}
					}
				}
			}
		}

		if len(results) != 0 {
			resultBundle := FeedResultBundle{
				Feed:  feedConfigItem,
				Items: results,
			}

			mainChannel <- &resultBundle
			// go persistResults(&resultBundle)
		}

		previous = current
		current = make(map[string]bool)
		results = make(map[string]*FeedResult)
	}
}

func (h *Hub) Run(sources *[]*FeedConfigItem) {
	feedChannel := make(chan *FeedResultBundle)
	timeout := 5

	for _, feedItem := range *sources {
		itemHandler := spawnItemHandler(feedChannel, feedItem, timeout)
		feed := rss.New(timeout, true, nil, itemHandler)

		go func() {
			for {
				if err := feed.Fetch(feedItem.Url, nil); err != nil {
					fmt.Fprintf(os.Stderr, "[e] %s: %s", feedItem.Url, err)
					return
				}

				<-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
			}
		}()
	}

	for {
		select {
		case connection := <-h.Register:
			h.Connections[connection] = true
		case connection := <-h.Unregister:
			delete(h.Connections, connection)
		case feedResults := <-feedChannel:
			for connection := range h.Connections {
				err := websocket.JSON.Send(connection, feedResults.Items)
				if err != nil {
					delete(h.Connections, connection)
					go connection.Close()
				}
			}
		}
	}
}
