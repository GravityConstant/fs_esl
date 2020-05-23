package dialplan

import (
	"bufio"
	"bytes"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/0x19/goesl"
)

type Request struct {
	Caller   string
	Callee   string
	Header   map[string]string
	Body     map[string]string
	URL      *url.URL
	IvrDepth []string          // ivr层深和存储ivr按键
	Params   map[string]string // url里带过来的参数解析
}

type Response struct {
	Conn *goesl.SocketConnection
}

// 模仿net/http包获取response和request
func GetReqRes(msg *goesl.Message, conn *goesl.SocketConnection) (Response, *Request) {
	r := &Request{
		Header:   msg.Headers,
		Body:     map[string]string{},
		URL:      &url.URL{},
		IvrDepth: []string{},
		Params:   map[string]string{},
	}
	r.parseTextBody(msg.Body)
	if strings.Compare(r.Header["Answer-State"], "ringing") == 0 {
		r.Caller = msg.Headers["Caller-Caller-Id-Number"]
		r.Callee = msg.Headers["Caller-Destination-Number"]
	} else {
		r.Caller = r.Body["Caller-Caller-Id-Number"]
		r.Callee = r.Body["Caller-Destination-Number"]
	}
	r.getRequest()

	w := Response{
		Conn: conn,
	}
	return w, r
}

func (r *Request) getRequest() {
	// outbound模式，响铃进来的电话处理
	if strings.Compare(r.Header["Answer-State"], "ringing") == 0 {
		// call in
		if t, ok := DialplanMap.Load(r.Callee); ok {
			if d, ok := t.(Dialplan); ok && d.Enabled {
				d.Blacklist.Caller = r.Caller
				FilterWrap(&d) // 进行黑名单，时间，区域设置
				goesl.Debug("dialplan_map: %#v", d)
				if d.Enabled {
					switch d.Params.(type) {
					case DirectDial:
						r.URL.Path = "/call/"
					case Ivr:
						r.URL.Path = "/ivr/true"
					default:
						r.URL.Path = "/busy/"
					}
				} else {
					r.URL.Path = "/busy/"
				}
			}
		}

	} else {
		// event body handle
		eventName := r.Body["Event-Name"]
		switch eventName {
		case "CHANNEL_EXECUTE_COMPLETE":
			appName := r.Body["Application"]
			if strings.Compare(appName, "play_and_get_digits") == 0 {
				// 获取ivr按键
				digits := r.Body["Variable_foo_dtmf_digits"]
				r.URL.Path = "/ivr/false"
				if strings.Compare(digits, "*") == 0 {
					// *号键返回上一级
					last := len(r.IvrDepth) - 1
					if last < 0 {
						last = 0
					}
					r.IvrDepth = r.IvrDepth[:last]
					return
				}
				// 按完键添加一级
				r.IvrDepth = append(r.IvrDepth, r.Body["Variable_foo_dtmf_digits"])
			}
		}
	}
}

// 解析消息体
func (r *Request) parseTextBody(bs []byte) {
	if len(bs) == 0 {
		goesl.Warning("parse text body empty.")
		return
	}

	buf := bufio.NewReader(bytes.NewReader(bs))
	cmr, err := textproto.NewReader(buf).ReadMIMEHeader()
	if err != nil {
		goesl.Error("parse text body MIME Header error: %v", err)
		return
	}
	for k, v := range cmr {

		r.Body[k] = v[0]

		// Will attempt to decode if % is discovered within the string itself
		if strings.Contains(v[0], "%") {
			r.Body[k], err = url.QueryUnescape(v[0])

			if err != nil {
				goesl.Error("parse text body error: %v", err)
				continue
			}
		}
	}
	// fmt.Printf("body================================================\n%v\n", r.Body)
}
