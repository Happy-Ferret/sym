// Code generated by "stringer -linecomment -type Base"; DO NOT EDIT.

package sym

import "strconv"

const _Base_name = "NULLVOIDCHARSHORTINTLONGFLOATDOUBLESTRUCTUNIONENUMMOEUCHARUSHORTUINTULONG"

var _Base_index = [...]uint8{0, 4, 8, 12, 17, 20, 24, 29, 35, 41, 46, 50, 53, 58, 64, 68, 73}

func (i Base) String() string {
	if i >= Base(len(_Base_index)-1) {
		return "Base(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Base_name[_Base_index[i]:_Base_index[i+1]]
}
