package client

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amrav/sparrow/proto"
	"github.com/fatih/color"
)

type Client struct {
	Active struct {
		Ip      net.IP
		Port    int
		UdpPort int
	}
	activeListener net.Listener
	hubConn        net.Conn
	hubListeners   chan listener

	// clientListeners is a locked map from nick to the client listeners
	// for that nick.
	// "*" is a special key, whose listeners will get all active connection
	// messages.
	clientListeners struct {
		sync.RWMutex
		m map[string]chan clientListener
	}
	User          proto.User
	outbox        chan outboxMsg
	clientUpdates chan clientUpdate
}

type outboxMsg struct {
	To  string
	Msg string
}

type clientUpdate struct {
	Nick   string
	Update string
	Conn   net.Conn
}

type connClient struct {
	Conn      net.Conn
	Listeners []listener
}

type listener struct {
	Messages chan []byte
	Done     chan struct{}
	Regex    *regexp.Regexp
}

type clientListener struct {
	Messages chan interface{}
	Done     chan struct{}
}

func New() *Client {
	c := &Client{
		hubListeners:  make(chan listener, 1000),
		outbox:        make(chan outboxMsg, 1000),
		clientUpdates: make(chan clientUpdate, 1000),
		clientListeners: struct {
			sync.RWMutex
			m map[string]chan clientListener
		}{m: make(map[string]chan clientListener)},
	}
	c.clientListeners.m["*"] = make(chan clientListener, 1000)
	c.User.ShareSize = 95 * 1024 * 1024 * 1024
	go c.transmit()
	return c
}

func sendClient(conn net.Conn, nick string, msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	magenta := color.New(color.FgMagenta).SprintfFunc()
	_, err := conn.Write([]byte(msg))
	if err != nil {
		log.Fatal("Error sending message to client: ", nick, ": ", err)
	}
	log.Print(magenta("client -> %s: ", nick), msg)
}

func (c *Client) transmit() {
	clientMsgs := make(map[string]chan string)
	hubMsgs := make(chan string, 1000)
	yellow := color.New(color.FgYellow).SprintFunc()

	// Run hub transmitter
	go func() {
		for m := range hubMsgs {
			_, err := c.hubConn.Write([]byte(m))
			if err != nil {
				log.Fatal("Couldn't write to hub: ", err)
			}
			log.Print(yellow("Client: "), m)
		}
	}()

	for {
		select {
		case m := <-c.outbox:
			if m.To == "" {
				hubMsgs <- m.Msg
			} else {
				if _, ok := clientMsgs[m.To]; !ok {
					clientMsgs[m.To] = make(chan string, 1000)
				}
				clientMsgs[m.To] <- m.Msg
			}

		case u := <-c.clientUpdates:
			switch u.Update {
			case "connected":
				if _, ok := clientMsgs[u.Nick]; !ok {
					clientMsgs[u.Nick] = make(chan string, 1000)
				}
				// Run transmitter per connected client
				go func(ch chan string) {
					for msg := range ch {
						sendClient(u.Conn, u.Nick, msg)
					}
				}(clientMsgs[u.Nick])
			case "disconnected":
				close(clientMsgs[u.Nick])
				delete(clientMsgs, u.Nick)
			}
		}
	}
}

func (c *Client) SetNick(nick string) {
	c.User.Nick = nick
	log.Print("Changed nick: ", c.User.Nick)
}

func (c *Client) StartActiveMode() {
	// Start TCP server
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("Failed to start active mode: ", err)
	}
	c.activeListener = ln
	c.Active.Port = ln.Addr().(*net.TCPAddr).Port

	// Get LAN IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal("Couldn't get interface addresses: ", err)
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		ipv4 := ip.To4()
		// Get machine IP of the form 10.x.x.x
		if ipv4 != nil && ipv4[0] == 10 {
			c.Active.Ip = ip
			break
		}
	}
	if c.Active.Ip == nil {
		log.Fatal("Couldn't find machine IP of form 10.x.x.x")
	}

	// Start UDP server
	addr := net.UDPAddr{
		Port: 0,
		IP:   c.Active.Ip,
	}
	udpConn, err := net.ListenUDP("udp", &addr)
	c.Active.UdpPort = udpConn.LocalAddr().(*net.UDPAddr).Port

	// Handle active connections
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Fatal("Couldn't accept connection: ", err)
			}
			go c.handleActiveConn(conn)
		}
	}()

	// Handle UDP packets
	go func() {
		// Send UDP messages to only *-subscribed client
		// listeners for now.
		// TODO: Clean up subscription code; stop using
		// channels as state queues
		all_cls := c.clientListeners.m["*"]
		for {
			buf := make([]byte, 2048)
			_, _, _ = udpConn.ReadFromUDP(buf)
			if err != nil {
				log.Fatalf("Couldn't read UDP packet: %s", err)
			}
			// log.Printf("Received UDP packet from %s: %s\n", addr, string(buf[0:n]))
			publishToListeners(string(buf), all_cls)
		}
	}()
}

// TODO: This function is NOT goroutine-safe. Calling it from multiple goroutines
// may lead to messages not reaching all subscribers, because multiple instances
// have pulled out listeners from the clientListener channel concurrently.
func publishToListeners(msg interface{}, listeners chan clientListener) {
	var cls []clientListener
loop:
	for {
		select {
		case l := <-listeners:
			cls = append(cls, l)
		default:
			break loop
		}
	}

	for _, cl := range cls {
		select {
		case <-cl.Done:
			close(cl.Messages)
			log.Print("Client listener sent close")
		case cl.Messages <- msg:
			listeners <- cl
		case <-time.After(1 * time.Second):
			log.Fatal("Error: wasn't able to write to client listener; dropping message after 2 seconds")
		}
	}
}

func (c *Client) handleActiveConn(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	otherNick := remote
	log.Print("Handling connection from: ", remote)

	blue := color.New(color.FgBlue).SprintfFunc()

	sendClient(conn, remote, "$MyNick %s|", c.User.Nick)

	reader := bufio.NewReader(conn)
	var cls chan clientListener
	all_cls := c.clientListeners.m["*"]

	for {
		msg, err := reader.ReadString('|')
		if err != nil {
			log.Fatal("Error reading from TCP connection: ",
				remote, " : ", err)
		}
		log.Print(blue("%s -> client: ", otherNick), msg)
		if strings.HasPrefix(msg, "$MyNick ") {
			otherNick = strings.Fields(msg)[1]
			otherNick = otherNick[:len(otherNick)-1]
			c.clientUpdates <- clientUpdate{
				Nick:   otherNick,
				Update: "connected",
				Conn:   conn,
			}

			c.clientListeners.Lock()
			var ok bool
			cls, ok = c.clientListeners.m[otherNick]
			if !ok {
				c.clientListeners.m[otherNick] = make(chan clientListener, 1000)
				cls = c.clientListeners.m[otherNick]
			}
			c.clientListeners.Unlock()

			sendClient(conn, otherNick,
				"$Lock EXTENDEDPROTOCOL/wut? Pk=gdc,Ref=10.109.49.49:411|")
		}
		if strings.HasPrefix(msg, "$Lock") {
			sendClient(conn, otherNick,
				"$Supports MiniSlots XmlBZList ADCGet TTHL TTHF|")
			sendClient(conn, otherNick, "$Direction Download 29000|")
			sendClient(conn, otherNick,
				"$Key %s|", proto.LockToKey(strings.Fields(msg)[1]))
		}

		publishToListeners(msg, cls)
		publishToListeners(msg, all_cls)

		// Check if we need to download something
		if strings.HasPrefix(msg, "$ADCSND file") {
			fields := strings.Fields(msg)
			numBytes, err := strconv.Atoi(fields[4][0 : len(fields[4])-1])
			if err != nil {
				log.Fatal("Error parsing numBytes: ", err)
			}
			log.Printf("Downloading file: %d bytes", numBytes)
			buf := make([]byte, numBytes)
			_, err = io.ReadFull(reader, buf)
			if err != nil {
				log.Fatal("Couldn't download something: ", err)
			}
			log.Print("Finished downloading file")
			publishToListeners(buf, cls)
			publishToListeners(buf, all_cls)

			// Remove the listener queue from nick-listener map
			// so it can be GC'd
			c.clientListeners.Lock()
			delete(c.clientListeners.m, otherNick)
			c.clientListeners.Unlock()
			// Close the all the listeners' channels to notify
			// them we're done
		loop:
			for {
				select {
				case l := <-cls:
					close(l.Messages)
				default:
					break loop
				}
			}
			// We're done with this active connection
			return
		}
	}
}

func (c *Client) Connect(hubAddr string) {
	log.Print("Username: ", c.User.Nick)
	log.Print("Connecting to hub: ", hubAddr)
	conn, err := net.DialTimeout("tcp", hubAddr, 5*time.Second)
	if err != nil {
		log.Fatal("Failed to connect to hub: ", err)
	}
	c.hubConn = conn
	done := make(chan struct{})
	msg := c.HubMessages(done)
	defer close(done)

	go c.handleHubMessages()

	for m := range msg {
		ms := string(m)
		if strings.HasPrefix(ms, "$Lock ") {
			lock := strings.Fields(ms)[1]
			key := proto.LockToKey(lock)
			c.MessageHub("$Key %s|", key)
			//c.MessageHub("$Lock EXTENDEDPROTOCOL/wut? Pk=gdcRef=10.109.49.49|")
			c.MessageHub("$ValidateNick %s|", c.User.Nick)
			//c.MessageHub("$Supports NoGetINFO NoHello UserIP2|")

			c.MessageHub("$Version 1,0091|")
			c.MessageHub(fmt.Sprintf("$MyINFO $ALL %s <gdc V:0.0.0,M:A,H:1/0/0,S:3>$ $10^Q$$%d$|", c.User.Nick, c.User.ShareSize))
		}
		if strings.HasPrefix(ms, "$Hello ") {
			break
		}
	}
}

type timedConn struct {
	*net.TCPConn
}

func (c *timedConn) Read(b []byte) (int, error) {
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	return c.TCPConn.Read(b)
}

func (c *Client) handleHubMessages() {
	yellow := color.New(color.FgYellow).SprintfFunc()
	reader := bufio.NewReader(&timedConn{c.hubConn.(*net.TCPConn)})
	for {
		msg, err := reader.ReadBytes(byte('|'))
		if err != nil {
			log.Fatal("Failed reading from hub: ", err)
		}
		// Comment out for less verbosity
		cyan := color.New(color.FgCyan).SprintFunc()
		log.Print(cyan("Hub: "), html.UnescapeString(string(msg)))
		var hls []listener
	loop:
		for {
			select {
			case hl := <-c.hubListeners:
				hls = append(hls, hl)
			default:
				break loop
			}
		}
		// TODO: hub listener only removed when message arrives
		// from hub and done is closed. Need to unsubscribe async.
		for _, hl := range hls {
			select {
			case <-hl.Done:
				close(hl.Messages)
				log.Print("Hub listener sent close")
				continue
			default:
			}
			if hl.Regex == nil || hl.Regex.Match(msg) {
				select {
				case hl.Messages <- msg:
					c.hubListeners <- hl
				case <-hl.Done:
					close(hl.Messages)
					log.Print("Hub listener sent close")
				default:
					log.Fatal(yellow("Warning: "),
						"Unable to write to hub listener, dropping message: ", string(msg))
				}
			} else {
				c.hubListeners <- hl
			}
		}
	}
}

func (c *Client) MessageHub(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	c.outbox <- outboxMsg{"", msg}
}

func (c *Client) MsgClient(nick string, msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	c.outbox <- outboxMsg{nick, msg}
}

func (c *Client) HubMessagesMatch(done chan struct{}, re *regexp.Regexp) chan []byte {
	ch := make(chan []byte, 1000)
	select {
	case c.hubListeners <- listener{ch, done, re}:
	default:
		panic("Tried adding too many hub listeners")
	}
	log.Print("Queued adding hub listener")
	return ch
}

func (c *Client) HubMessages(done chan struct{}) chan []byte {
	return c.HubMessagesMatch(done, nil)
}

func (c *Client) ClientMessagesMatch(done chan struct{}, re *regexp.Regexp) chan []byte {
	ch := make(chan []byte, 1000)
	select {
	case c.hubListeners <- listener{ch, done, re}:
	default:
		panic("Tried adding too many hub listeners")
	}
	log.Print("Queued adding hub listener")
	return ch
}

func (c *Client) ClientMessages(nick string, done chan struct{}) chan interface{} {
	ch := make(chan interface{}, 100)
	l := clientListener{ch, done}

	c.clientListeners.Lock()
	clc, ok := c.clientListeners.m[nick]
	if !ok {
		c.clientListeners.m[nick] = make(chan clientListener, 1000)
		clc = c.clientListeners.m[nick]
	}
	c.clientListeners.Unlock()
	select {
	case clc <- l:
	default:
		panic("Tried adding too many client listeners")
	}

	log.Print("Added client listener")
	return ch
}
