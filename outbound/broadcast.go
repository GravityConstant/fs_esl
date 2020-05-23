package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/textproto"
	"net/url"
	"runtime"
	"sort"
	"strings"

	. "github.com/0x19/goesl"
)

var (
	common_ring_path   = "/home/voices/rings/common/"
	specific_ring_path = "/home/voices/rings/uploads/"
	record_path        = "/home/voices/records/"
	file               = specific_ring_path + "4000400426/20190122144123.wav"
	data               = "{sip_h_Diversion=<sip:28324285@ip>}[leg_timeout=20]sofia/gateway/zqzj/13860661577"
)

func main() {

	defer func() {
		if r := recover(); r != nil {
			Error("Recovered in f", r)
		}
	}()

	// Boost it as much as it can go ...
	runtime.GOMAXPROCS(runtime.NumCPU())
	// server
	if s, err := NewOutboundServer(":8484"); err != nil {
		Error("Got error while starting Freeswitch outbound server: %s", err)
	} else {
		go handle(s)
		s.Start()
	}

}

// handle - Running under goroutine here to explain how to handle playback ( play to the caller )
func handle(s *OutboundServer) {

	for {

		select {

		case conn := <-s.Conns:
			eventMsg(&conn)
			Notice("New incomming connection: %v", conn)

			if err := conn.Connect(); err != nil {
				Error("Got error while accepting connection: %s", err)
				break
			} else {
				conn.Send("event plain CHANNEL_CALLSTATE")
				conn.Execute("bridge", data, false)
			}

		default:
			// YabbaDabbaDooooo!
			// Flintstones. Meet the Flintstones. They're the modern stone age family. From the town of Bedrock,
			// They're a page right out of history. La la,lalalalala la :D
		}
	}

}

func eventMsg(conn *SocketConnection) {
	go func() {
		for {
			msg, err := conn.ReadMessage()

			if err != nil {

				// If it contains EOF, we really dont care...
				if !strings.Contains(err.Error(), "EOF") {
					Error("Error while reading Freeswitch message: %s", err)
				}
				break
			}

			body := parseTextBody(msg.Body)
			// Debug("%s", msg)
			// if body["Event-Name"] == "CHANNEL_CALLSTATE" {
			// 	PrettyPrint(body)
			// }

			if body["Event-Name"] == "CHANNEL_CALLSTATE" &&
				body["Answer-State"] == "answered" &&
				body["Caller-Direction"] == "inbound" {

				conn.BgApi(fmt.Sprintf("uuid_broadcast %s %s both", body["Other-Leg-Unique-Id"], file))
			}
		}
	}()
}

// PrettyPrint prints Event headers and body to the standard output.
func PrettyPrint(msg map[string]string) {
	var keys []string
	for k := range msg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s: %s\n", k, msg[k])
	}
}

func parseTextBody(bs []byte) map[string]string {
	res := make(map[string]string)

	if len(bs) == 0 {
		Warning("parse text body empty.")
		return res
	}

	buf := bufio.NewReader(bytes.NewReader(bs))
	cmr, err := textproto.NewReader(buf).ReadMIMEHeader()
	if err != nil {
		Error("parse text body MIME Header error: %v", err)
		return res
	}
	for k, v := range cmr {

		res[k] = v[0]

		// Will attempt to decode if % is discovered within the string itself
		if strings.Contains(v[0], "%") {
			res[k], err = url.QueryUnescape(v[0])

			if err != nil {
				Error("parse text body error: %v", err)
				continue
			}
		}
	}
	return res
}
