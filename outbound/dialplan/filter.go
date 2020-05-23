package dialplan

import (
	"strconv"
	"strings"
)

type Filter interface {
	BlacklistFilter()
	TimeSetFilter()
	AreaSetFilter()
}

type BaseFilter struct {
	Blacklist Blacklist
	TimeSet   TimeSet
	AreaSet   AreaSet
}

type Blacklist struct {
	Caller  string
	Param   []string
	Operand string
}

type TimeSet struct {
	TimeSeg map[string]string
	Operand string
}

type AreaSet struct {
	DistrictNo []string
	Operand    string
}

func FilterWrap(f Filter) {
	f.BlacklistFilter()
	f.TimeSetFilter()
	f.AreaSetFilter()
}

func (bf *BaseFilter) isInBlacklist() bool {
	for _, val := range bf.Blacklist.Param {
		if strings.Compare(bf.Blacklist.Caller, val) == 0 {
			return true
		}
	}
	return false
}

func (bf *BaseFilter) isInTime() bool {
	return true
}

func (bf *BaseFilter) isInArea() bool {
	return true
}

// -1是一个无效值
func GetIntTimeBySegment(s string) (start, end int) {
	start, end = -1, -1
	if !strings.Contains(s, "-") {
		if i, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			start = i
		}
	} else {
		sArr := strings.Split(s, "-")
		if i, err := strconv.Atoi(strings.TrimSpace(sArr[0])); err == nil {
			start = i
		}
		if i, err := strconv.Atoi(strings.TrimSpace(sArr[1])); err == nil {
			if start == -1 {
				start = i
			} else {
				end = i
			}
		}
	}

	return
}
