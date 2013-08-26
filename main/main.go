package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/tumas/threaded"
	"net/http"
	"text/template"
)

var homeTemplate = template.Must(template.ParseFiles("home.html"))

func homeHandler(c http.ResponseWriter, req *http.Request) {
	homeTemplate.Execute(c, req.Host)
}

func main() {
	sources := []*threaded.FeedConfigItem{
		&threaded.FeedConfigItem{
			Guid: "geras_dviratis",
			Url:  "http://www.gerasdviratis.lt/forum/syndication.php",
			Identifier: &threaded.FeedIdentifier{
				ParamName: "t",
				ParamType: "parameter",
			},
		},
	}

	hub := threaded.Hub{
		Connections: make(map[*websocket.Conn]bool),
		Register:    make(chan *websocket.Conn),
		Unregister:  make(chan *websocket.Conn),
	}

	wsHandler := func(ws *websocket.Conn) {
		hub.Register <- ws
		// defer func() { hub.unregister <- ws }()
		select {}
	}

	go hub.Run(&sources)

	http.HandleFunc("/", homeHandler)
	http.Handle("/ws", websocket.Handler(wsHandler))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("ListenAndServe: %s", err)
	}
}
