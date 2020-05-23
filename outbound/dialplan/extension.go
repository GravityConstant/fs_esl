package dialplan

import (
	"sync"
)

var DialplanMap sync.Map

type Dialplan struct {
	BaseFilter
	Name    string
	Number  string
	Enabled bool
	Params  interface{}
}

// 根据不同的主叫设置执行不同的方式
// enabled：是否启用这个呼叫路由
// Direct：直接直转
// Ivr：转ivr
func (d *Dialplan) BlacklistFilter() {
	d.Blacklist.Param = []string{"83127866"}
	d.Blacklist.Operand = "Enabled"
	if d.isInBlacklist() {
		switch d.Blacklist.Operand {
		case "Enabled":
			d.Enabled = false
		case "Direct":
			d.Params = InitDirectDial()
		case "Ivr":
			d.Params = InitIvrMenu()
		}
	} else {
		switch d.Blacklist.Operand {
		case "Enabled":
			d.Enabled = true
			d.Params = InitDirectDial()
		case "Direct":
			d.Params = InitDirectDial()
		case "Ivr":
			d.Params = InitIvrMenu()
		}
	}
	DialplanMap.Store(d.Number, *d)
}

// 根据时间选择路由方式
func (d *Dialplan) TimeSetFilter() {
	// d.Time = map[string]string{
	// 	"minute": "0-14",
	// }
	// for key, seg := range d.Time {
	// 	switch key {
	// 	case "wday":

	// 	case "hour":

	// 	case "minute":

	// 	}
	// }
}

// 根据主叫号码选择路由方式
func (d *Dialplan) AreaSetFilter() {

}

type DirectDial struct {
	ResponseType     int // 0顺序接听，1随机接听
	IgnoreEarlyMedia bool
	RingPath         string
	BindPhone        []*BindPhone
}

type BindPhone struct {
	LegTimeout int
	Name       string
	Gateway    string // internal:内线，其他视为外线
}

func InitDirectDial() DirectDial {
	bp := &BindPhone{
		LegTimeout: 20,
		Name:       "user/1000",
		Gateway:    "internal",
	}
	bps := []*BindPhone{bp}

	dd := DirectDial{
		ResponseType:     0,
		IgnoreEarlyMedia: true,
		RingPath:         "/home/voices/rings/common/ring.wav",
		BindPhone:        bps,
	}

	return dd
}

// 初始化数据，数据后期来自数据库
func InitDialplanMap() {
	d := Dialplan{
		Name:    "test",
		Number:  "123456",
		Enabled: true,
	}

	/*
	 * --------------------------------------------------------------
	 */
	d2 := Dialplan{
		Name:    "4000400426",
		Number:  "123456",
		Enabled: true,
	}

	DialplanMap.Store(d.Number, d)
	DialplanMap.Store(d2.Number, d2)
}

type Ivr struct {
	Root *Menu
}

// play_and_get_digits的参数
type Menu struct {
	Name         string //
	Min          int    // Minimum number of digits to fetch (minimum value of 0)
	Max          int    // Maximum number of digits to fetch (maximum value of 128)
	Tries        int    // Number of tries for the sound to play
	Timeout      int    // Number of milliseconds to wait for a dialed response after the file playback ends and before PAGD does a retry.
	Terminators  string // digits used to end input if less than <max> digits have been pressed. If it starts with '=', then a terminator must be present for the input to be accepted (Since FS 1.2). (Typically '#', can be empty). Add '+' in front of terminating digit to always append it to the result variable specified in var_name.
	File         string // Sound file to play to prompt for digits to be dialed by the caller; playback can be interrupted by the first dialed digit (can be empty or the special string "silence" to omit the message).
	InvalidFile  string // Sound file to play when digits don't match the regexp (can be empty to omit the message).
	VarName      string // Channel variable into which valid digits should be placed (optional, no variable is set by default. See also 'var_name_invalid' below).
	Regexp       string // Regular expression to match digits (optional, an empty string allows all input (default)).
	DigitTimeout int    // Inter-digit timeout; number of milliseconds allowed between digits in lieu of dialing a terminator digit; once this number is reached, PAGD assumes that the caller has no more digits to dial (optional, defaults to the value of <timeout>).
	Child        map[string]interface{}
}

// 根据freeswitch的ivr按键完构建的数据结构
type Entry struct {
	App  string
	Data string
}

// 初始化ivr，后期从数据库取值
func InitIvrMenu() Ivr {
	ivr := Ivr{
		Root: &Menu{
			Name:         "40004004261000",
			Min:          1,
			Max:          1,
			Tries:        3,
			Timeout:      3000,
			Terminators:  "#",
			File:         `/home/voices/rings/uploads/4000400426/20190122150848.wav`,
			InvalidFile:  `/home/voices/rings/common/input_error.wav`,
			VarName:      `foo_dtmf_digits`,
			Regexp:       `\d`,
			DigitTimeout: 3000,
			Child: map[string]interface{}{
				"1": Entry{"playback", "/home/voices/rings/common/busy.wav"},
				"2": Menu{
					Name:         "40004004261001",
					Min:          1,
					Max:          1,
					Tries:        3,
					Timeout:      3000,
					Terminators:  "#",
					File:         `/home/voices/rings/uploads/4000400426/20190124171007.wav`,
					InvalidFile:  `/home/voices/rings/common/input_error.wav`,
					VarName:      `foo_dtmf_digits`,
					Regexp:       `\d|\*`,
					DigitTimeout: 3000,
				},
			},
		},
	}

	return ivr

	// DialplanMap.Store("123456", ivr)
}

// depth长度代表ivr的层数，其中的数字代表按键
// depth为0返回ivr的根目录
// depth长度和层深一致时，返回此时的ivr
// 长度不一致时，继续搜索下层ivr
// 但是碰到entry时，为要执行的freeswitch app
func (ivr Ivr) FindIvrMenu(depth []string) interface{} {
	depthLen := len(depth)
	if depthLen == 0 {
		return *ivr.Root
	}

	t := ivr.Root.Child
	for k, v := range depth {
		if i, ok := t[v]; ok {
			switch tv := i.(type) {
			case Menu:
				if k+1 == depthLen {
					return tv
				}
				t = tv.Child
			case Entry:
				return tv
			}
		}
	}
	return Entry{"hangup", ""}
}
