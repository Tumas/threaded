package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	rss "github.com/jteeuwen/go-pkg-rss"
	"net/http"
	"net/url"
	"os"
	"text/template"
	"time"
)

type FeedIdentifier struct {
	paramName string
	paramType string
}

type FeedConfigItem struct {
	guid       string
	url        string
	identifier *FeedIdentifier
}

type FeedResultBundle struct {
	feed  *FeedConfigItem
	items map[string][]*rss.Item
}

type hub struct {
	connections map[*websocket.Conn]bool
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
}

func spawnItemHandler(mainChannel chan *FeedResultBundle, feedConfigItem *FeedConfigItem, timeout int) func(*rss.Feed, *rss.Channel, []*rss.Item) {
	previous := make(map[string]bool)
	current := make(map[string]bool)
	results := make(map[string][]*rss.Item)

	return func(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
		ident := feedConfigItem.identifier

		for _, item := range newitems {
			u, err := url.Parse(item.Guid)
			if err != nil {
				fmt.Printf("Error when parsing guid: %s\n", item.Guid)
				continue
			}

			if ident.paramType == "parameter" {
				queryParams, err := url.ParseQuery(u.RawQuery)
				if err != nil {
					fmt.Printf("Error when parsing query params: %s\n", u.RawQuery)
					continue
				}

				branch := queryParams[ident.paramName][0]
				// if branch, ok := queryParams[ident.paramName][0]; !ok {
				// 	fmt.Printf("Could not find topic id! Config mismatch!")
				// 	continue
				// }

				current[item.Guid] = true

				if _, ok := previous[item.Guid]; !ok {
					if array, ok := results[branch]; ok {
						results[branch] = append(array, item)
					} else {
						results[branch] = []*rss.Item{item}
					}
				}
			}
		}

		if len(results) != 0 {
			mainChannel <- &FeedResultBundle{
				feed:  feedConfigItem,
				items: results,
			}
		}

		previous = current
		current = make(map[string]bool)
		results = make(map[string][]*rss.Item)
	}
}

func (h *hub) run(sources *[]*FeedConfigItem) {
	feedChannel := make(chan *FeedResultBundle)
	timeout := 5

	for _, feedItem := range *sources {
		itemHandler := spawnItemHandler(feedChannel, feedItem, timeout)
		feed := rss.New(timeout, true, nil, itemHandler)

		go func() {
			for {
				if err := feed.Fetch(feedItem.url, nil); err != nil {
					fmt.Fprintf(os.Stderr, "[e] %s: %s", feedItem.url, err)
					return
				}

				<-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
			}
		}()
	}

	for {
		select {
		case connection := <-h.register:
			h.connections[connection] = true
		case connection := <-h.unregister:
			delete(h.connections, connection)
		case feedResults := <-feedChannel:
			for connection := range h.connections {
				err := websocket.JSON.Send(connection, feedResults.items)
				if err != nil {
					delete(h.connections, connection)
					go connection.Close()
				}
			}
		}
	}
}

var homeTemplate = template.Must(template.ParseFiles("home.html"))

func homeHandler(c http.ResponseWriter, req *http.Request) {
	homeTemplate.Execute(c, req.Host)
}

func main() {
	sources := []*FeedConfigItem{
		&FeedConfigItem{
			guid: "geras_dviratis",
			url:  "http://www.gerasdviratis.lt/forum/syndication.php",
			identifier: &FeedIdentifier{
				paramName: "t",
				paramType: "parameter",
			},
		},
	}

	hub := hub{
		connections: make(map[*websocket.Conn]bool),
		register:    make(chan *websocket.Conn),
		unregister:  make(chan *websocket.Conn),
	}

	wsHandler := func(ws *websocket.Conn) {
		hub.register <- ws
		// defer func() { hub.unregister <- ws }()
		select {}
	}

	go hub.run(&sources)

	http.HandleFunc("/", homeHandler)
	http.Handle("/ws", websocket.Handler(wsHandler))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("ListenAndServe: %s", err)
	}
}
