package agentsetup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// jval is a JSON value that keeps the order of an object's keys and the text of its numbers, so
// that adding one entry to a user's config file changes nothing else in it.
type jval struct {
	kind byte // 'o' object, 'a' array, 's' scalar
	keys []string
	vals []*jval
	raw  []byte // a scalar's JSON text
}

func parseJSON(b []byte) (*jval, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("more than one JSON value")
	}
	return v, nil
}

func parseValue(dec *json.Decoder) (*jval, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, isDelim := tok.(json.Delim)
	if !isDelim {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if n, ok := tok.(json.Number); ok {
			buf.WriteString(n.String())
		} else if err := enc.Encode(tok); err != nil {
			return nil, err
		}
		return &jval{kind: 's', raw: bytes.TrimSpace(buf.Bytes())}, nil
	}
	v := &jval{kind: 'a'}
	if d == '{' {
		v.kind = 'o'
	}
	for dec.More() {
		if v.kind == 'o' {
			kt, err := dec.Token()
			if err != nil {
				return nil, err
			}
			v.keys = append(v.keys, kt.(string))
		}
		child, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		v.vals = append(v.vals, child)
	}
	if _, err := dec.Token(); err != nil { // the closing bracket
		return nil, err
	}
	return v, nil
}

// get is the value of a key of an object, or nil.
func (v *jval) get(key string) *jval {
	for i, k := range v.keys {
		if k == key {
			return v.vals[i]
		}
	}
	return nil
}

// set adds key to an object, or replaces its value.
func (v *jval) set(key string, val *jval) {
	for i, k := range v.keys {
		if k == key {
			v.vals[i] = val
			return
		}
	}
	v.keys, v.vals = append(v.keys, key), append(v.vals, val)
}

// setPath puts val at path under the root object, making the objects on the way.
func (v *jval) setPath(path []string, val *jval) error {
	cur := v
	for i, key := range path {
		if i == len(path)-1 {
			cur.set(key, val)
			return nil
		}
		next := cur.get(key)
		switch {
		case next == nil:
			next = &jval{kind: 'o'}
			cur.set(key, next)
		case next.kind != 'o':
			return fmt.Errorf("%q is not an object", key)
		}
		cur = next
	}
	return nil
}

func (v *jval) compact(buf *bytes.Buffer) {
	switch v.kind {
	case 's':
		buf.Write(v.raw)
	case 'a':
		buf.WriteByte('[')
		for i, c := range v.vals {
			if i > 0 {
				buf.WriteByte(',')
			}
			c.compact(buf)
		}
		buf.WriteByte(']')
	default:
		buf.WriteByte('{')
		for i, c := range v.vals {
			if i > 0 {
				buf.WriteByte(',')
			}
			k, _ := json.Marshal(v.keys[i])
			buf.Write(k)
			buf.WriteByte(':')
			c.compact(buf)
		}
		buf.WriteByte('}')
	}
}

// render is the value as indented JSON, with indent as one level.
func (v *jval) render(indent string) []byte {
	var compact, out bytes.Buffer
	v.compact(&compact)
	if err := json.Indent(&out, compact.Bytes(), "", indent); err != nil {
		return compact.Bytes()
	}
	return out.Bytes()
}

// indentOf is the indentation a file uses: the whitespace of its first indented line, else two
// spaces.
func indentOf(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || len(trimmed) == len(line) {
			continue
		}
		return line[:len(line)-len(trimmed)]
	}
	return "  "
}
