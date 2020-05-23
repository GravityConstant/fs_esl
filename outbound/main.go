// Copyright 2015 Nevio Vesic
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

/*
event plain ALL
event plain CHANNEL_EXECUTE_COMPLETE

api status
api uuid_broadcast <uid> xxx.wav both

sendmsg
call-command: execute
execute-app-name: bridge
execute-app-arg: {origination_caller_id_number=28324285}sofia/gateway/zqzj/13675017141
event-lock: true/false

sendmsg
call-command: execute
execute-app-name: play_and_get_digits
execute-app-arg: 2 5 3 7000 # /home/voices/rings/common/tip_record.wav /home/voices/rings/common/input_error.wav foobar \d+
event-lock: true/false

sendmsg
call-command: execute
execute-app-name: playback
execute-app-arg: /home/voices/rings/common/ivr_transfer.wav
event-lock: true/false

year = 4 digit year. Example year="2009"
yday = 1-365
mon = 1-12
mday = 1-31
week = 1-52
mweek= 1-6
wday = 1-7
hour = 0-23
minute = 0-59
minute-of-day = 1-1440

*/

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
	"github.com/GravityConstant/fs_esl/outbound/dialplan"
)

var (
	common_ring_path   = "/home/voices/rings/common/"
	specific_ring_path = "/home/voices/rings/uploads/"
	record_path        = "/home/voices/records/"
)

func main() {

	defer func() {
		if r := recover(); r != nil {
			Error("Recovered in f", r)
		}
	}()

	// Boost it as much as it can go ...
	runtime.GOMAXPROCS(runtime.NumCPU())
	// init
	dialplan.InitDialplanMap()
	// server
	if s, err := NewOutboundServer(":8084"); err != nil {
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
				// conn.Execute("park", "", false)
				conn.Send("event plain CHANNEL_CREATE CHANNEL_EXECUTE_COMPLETE CHANNEL_ANSWER")
				// conn.Send("event plain ALL")
			}

			// answer, err := conn.ExecuteAnswer("", false)

			// if err != nil {
			// 	Error("Got error while executing answer: %s", err)
			// 	break
			// }

			// Debug("Answer Message: %s", answer)
			// Debug("Caller UUID: %s", answer.GetHeader("Caller-Unique-Id"))

			// cUUID := answer.GetCallUUID()

			// if sm, err := conn.Execute("playback", welcomeFile, true); err != nil {
			// 	Error("Got error while executing playback: %s", err)
			// 	break
			// } else {
			// 	Debug("Playback Message: %s", sm)
			// }

			// if hm, err := conn.ExecuteHangup(cUUID, "", false); err != nil {
			// 	Error("Got error while executing hangup: %s", err)
			// 	break
			// } else {
			// 	Debug("Hangup Message: %s", hm)
			// }

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

			// Debug("%s", msg)
			Debug("%s", "NEW MESSAGE INCOMING......")
			// PrettyPrint(msg)

			// get req, res
			w, r := dialplan.GetReqRes(msg, conn)
			dialplan.DefaultServeMux.ServeFreeswitch(w, r)
		}
	}()
}

// PrettyPrint prints Event headers and body to the standard output.
func PrettyPrint(msg *Message) {
	var keys []string
	for k := range msg.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s: %s\n", k, msg.Headers[k])
	}
	if len(msg.Body) > 0 {
		body, _ := url.QueryUnescape(string(msg.Body))
		fmt.Printf("BODY: \n%s\n", body)
		// parseTextBody(msg)
	}
}

// 解析消息体，这里只用于调试打印消息
func parseTextBody(msg *Message) error {
	var err error
	buf := bufio.NewReader(bytes.NewReader(msg.Body))
	var body textproto.MIMEHeader
	if body, err = textproto.NewReader(buf).ReadMIMEHeader(); err != nil {
		return fmt.Errorf("parse text body: %v", err)
	}
	var keys []string
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s: %s\n", k, body[k])
	}
	return err
}
