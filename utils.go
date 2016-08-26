package orm

import "bytes"

func snakeToUpperCamel(s string) string {
	buf := new(bytes.Buffer)
	first := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' && first {
			buf.WriteByte(c - 32)
			first = false
		} else if c == '_' {
			first = true
			continue
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}
