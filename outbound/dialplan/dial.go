package dialplan

import (
	"fmt"

	"github.com/0x19/goesl"
)

func call(w Response, r *Request) {
	w.Conn.Execute("bridge", "{origination_caller_id_number=83127866,sip_h_Diversion=<sip:28324284@ip>}sofia/gateway/zqzj/13675017141", false)

	if t, ok := DialplanMap.Load(r.Callee); ok {
		if d, ok := t.(Dialplan); ok {
			if dd, ok := d.Params.(DirectDial); ok {

			}
		}
	}
}

func ivr(w Response, r *Request) {
	goesl.Debug("request params: %#v, depth: %#v", r.Params, r.IvrDepth)
	if r.Params["first"] == "true" {
		w.Conn.Execute("answer", "", true)
	}

	if t, ok := DialplanMap.Load(r.Callee); ok {
		goesl.Debug("dialplan interface: %#v", t)
		if d, ok := t.(Dialplan); ok {
			goesl.Debug("dialplan: %#v", d)
			if ivr, ok := d.Params.(Ivr); ok {
				menu := ivr.FindIvrMenu(r.IvrDepth)
				goesl.Debug("menu: %#v", menu)
				switch m := menu.(type) {
				case Menu:
					data := `%d %d %d %d %s %s %s %s %s %d`
					data = fmt.Sprintf(data, m.Min, m.Max, m.Tries, m.Timeout, m.Terminators, m.File, m.InvalidFile, m.VarName, m.Regexp, m.DigitTimeout)
					w.Conn.Execute("play_and_get_digits", data, false)
				case Entry:
					w.Conn.Execute(m.App, m.Data, false)
				}

			}
		}
	}
}

func busy(w Response, r *Request) {
	w.Conn.Execute("respond", "486", false)
}
