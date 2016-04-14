package main

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"

	"github.com/amrav/sparrow/client"
	"github.com/amrav/sparrow/proto"
	"github.com/amrav/sparrow/server"
)

func main() {
	c := client.New()
	c.StartActiveMode()
	c.SetNick(proto.GenerateRandomUsername())
	c.Connect("10.109.49.49:411")

	s := server.New(c)
	s.Register("", SendHubMessages)
	s.Register("", SendPrivateMessages)
	s.Register("MAKE_SEARCH_QUERY", HandleSearchRequests)

	http.Handle("/connect", websocket.Handler(s.WsHandler))
	http.Handle("/", http.FileServer(http.Dir("ui")))
	log.Fatal(http.ListenAndServe(":12345", nil))
}
