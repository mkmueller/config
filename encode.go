// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"io"
	"os"
	"fmt"
	"sort"
	"time"
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// The Encoder handles encoding a struct to an io.Writer.
type Encoder struct {
	writer       io.Writer
	previous_key string
	options      int
	v            reflect.Value
	fileMode     os.FileMode
	errs         []error
}

// NewEncoder accepts a struct or map and returns a new Encoder.
func NewEncoder(x interface{}, options ...int) *Encoder {
	rv := reflect.ValueOf(x)
	switch rv.Kind() {
	case reflect.Ptr:
		if rv.Elem().Kind() == reflect.Struct {
			rv = rv.Elem()
			break
		}
		panic("Expecting a struct or a map")
	case reflect.Map:
		break
	case reflect.Struct:
		break
	default:
		panic("Expecting a struct or a map")
	}
	o := &Encoder{v: rv}
	if len(options) > 0 {
		if !o.allowedOption(options[0]) {
			panic("Option not allowed")
		}
		o.options = options[0]
	}
	return o
}

func (o *Encoder) allowedOption(option int) bool {
	return option == option&(ENCODE_ZERO_VALUES|ENCODE_LOWER_CASE|ENCODE_SNAKE_CASE|OVERWRITE_FILE)
}

// ToFile will encode a struct to the supplied filename. If the file exists,
// it will not be overwritten unless the overwrite options is used.
func (o *Encoder) ToFile(filename string) error {
	fi, err := os.Stat(filename)
	if err == nil {
		// file exists
		if fi.IsDir() {
			return errors.New("cannot overwrite a directory")
		}
		if OVERWRITE_FILE != OVERWRITE_FILE&(o.options) {
			return errors.New("file already exists")
		}
	}
	fh, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		fi, err := fh.Stat()
		fh.Close()
		if err == nil {
			// Remove file if it's empty
			if fi.Size() == 0 {
				os.Remove(filename)
			}
		}
	}()
	// We don't care if chmod returns an error. Just ignore it.
	fh.Chmod(o.fileMode)
	return o.ToStream(fh)
}

func Encode(x interface{}, options ...int) ([]byte, error) {
	o := NewEncoder(x, options...)
	var buf bytes.Buffer
	o.writer = &buf
	o.encodeTraverseStruct(o.v, 0, "")
	return buf.Bytes(), getErrors(o.errs)
}

func EncodeToFile(x interface{}, filename string, options ...int) error {
	return NewEncoder(x, options...).ToFile(filename)
}

// ToBytes
func (o *Encoder) ToBytes(bs *[]byte) error {
	var buf bytes.Buffer
	err := o.ToStream(&buf)
	*bs = buf.Bytes()
	return err
}

// ToStream
func (o *Encoder) ToStream(w io.Writer) error {
	o.writer = w
	o.encodeTraverseStruct(o.v, 0, "")
	return getErrors(o.errs)
}

func (o *Encoder) appendErr(s string, v interface{}) {
	o.errs = append(o.errs, errors.New(fmt.Sprintf(s, v)))
}

func (o *Encoder) encodeTraverseStruct(v1 reflect.Value, depth int, parent_key string) bool {
	switch v1.Kind() {
	case reflect.Map:
		return o.encodeMap(v1, depth, parent_key)
	case reflect.Struct:
		if isTimeType(v1.Type()) {
			return o.encodeTime(v1, depth, parent_key)
		}
		return o.encodeStruct(v1, depth, parent_key)
	default:
		if !o.encodeScalar(v1, depth, parent_key) {
			o.appendErr("Cannot encode type (%v)", v1.Kind())
			return false
		}
	}
	return true
}

func (o *Encoder) encodeTime(v1 reflect.Value, depth int, parent_key string) bool {
	if isTimeType(v1.Type()) {
		t := v1.Interface().(time.Time)
		var dt string
		switch {
		case isTimeOnly(t):
			dt = t.Format(time_fmt)
		case isDateOnly(t):
			dt = t.Format(date_fmt)
		case isDateTime(t):
			dt = t.Format(date_time)
		case isUTCTime(t):
			dt = t.Format(utc_time)
		case isUTCDate(t):
			dt = t.Format(utc_date)
		}
		o.write_kv(depth, parent_key, dt)
	}
	return true
}

func (o *Encoder) encodeScalar(v1 reflect.Value, depth int, parent_key string) bool {
	switch v1.Kind() {
	case reflect.String:
		o.encodeString(v1, depth, parent_key)
	case reflect.Bool:
		BoolStr := "False"
		if v1.Interface().(bool) == true {
			BoolStr = "True"
		}
		if !o.isOption(ENCODE_ZERO_VALUES) && !v1.Interface().(bool) {
			break
		}
		o.write_kv(depth, parent_key, BoolStr)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64:
		if !o.isOption(ENCODE_ZERO_VALUES) && isZero(v1) {
			break
		}
		o.write_kv(depth, parent_key, v1)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint, reflect.Uint64:
		if !o.isOption(ENCODE_ZERO_VALUES) && isZero(v1) {
			break
		}
		o.write_kv(depth, parent_key, v1)
	case reflect.Float32, reflect.Float64:
		if !o.isOption(ENCODE_ZERO_VALUES) && isZero(v1) {
			break
		}
		o.write_kv(depth, parent_key, v1)
	default:
		return false
	}
	return true
}

func (o *Encoder) encodeString(v1 reflect.Value, depth int, parent_key string) bool {
	str := v1.String()
	if len(str) > 50 {
		if needs_heredoc(str) {
			str = output_heredoc(str)
		} else {
			str = encodeMultiline(parent_key, str)
		}
	} else {
		str = quote(str)
	}
	if str == "" {
		if o.isOption(ENCODE_ZERO_VALUES) {
			str = `""`
		} else {
			return true
		}
	}
	o.write_kv(depth, parent_key, str)
	return true
}

func (o *Encoder) encodeMap(v1 reflect.Value, depth int, parent_key string) bool {
	last_parent := ""
	open__brace := false
	keys := v1.MapKeys()
	sorted := make([]string, len(keys))
	for i, k := range keys {
		sorted[i] = k.String()
	}
	sort.Strings(sorted)
	for _, ky := range sorted {
		this_key := ky
		v := v1.MapIndex(reflect.ValueOf(ky))
		if !(o.isOption(ENCODE_ZERO_VALUES) && isZeroStruct(v1)) {
			if parent_key != o.previous_key && last_parent != parent_key {
				o.previous_key = parent_key
				o.write_kv(depth, parent_key, "{")
				open__brace = true
				last_parent = parent_key
			}
			o.encodeTraverseStruct(v, depth+1, this_key)
		}
	}
	if open__brace && parent_key != "" {
		o.write(depth, "}\n")
		open__brace = false
	}
	return true
}

func (o *Encoder) encodeStruct(v1 reflect.Value, depth int, parent_key string) bool {
	last_parent := ""
	open__brace := false
	for i, n := 0, v1.NumField(); i < n; i++ {
		this_key := v1.Type().Field(i).Name
		if !isPublic(this_key) {
			continue
		}
		if parent_key != "" {
			if !o.isOption(ENCODE_ZERO_VALUES) && isZeroStruct(v1) {
				continue
			}
			if parent_key != o.previous_key && last_parent != parent_key {
				o.previous_key = parent_key
				o.write_kv(depth, parent_key, "{")
				open__brace = true
				last_parent = parent_key
			}
		}
		if !o.encodeTraverseStruct(v1.Field(i), depth+1, this_key) {
			continue
		}
	}
	if open__brace && parent_key != "" {
		o.write(depth, "}\n")
		open__brace = false
	}
	return true
}

func isZero(v reflect.Value) bool {
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func isZeroStruct(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Struct:
		if isTimeType(v.Type()) {
			return isZero(v)
		}
		r := true
		for i := 0; i < v.NumField(); i++ {
			if isPublic(v.Type().Field(i).Name) {
				r = r && isZeroStruct(v.Field(i))
			}
		}
		return r
	}
	return isZero(v)
}

// Break long lines at word boundaries
func encodeMultiline(parent_key, str string) string {
	var ar []string
	var i, n int
	width := multi_line_width - (len(parent_key) + 3)
	for {
		n = i + width
		if n >= len(str) {
			break
		}
		x := strings.LastIndex(str[i:n], " ")
		if x > -1 {
			x += i + 1
			n = x
			// if we landed on a space, find next non-space

			for n < len(str) {
				if str[n:n+1] == " " {
					break
				}
				n++
			}
		}
		ar = append(ar, quote(str[i:n]))
		i = n
	}
	ar = append(ar, quote(str[i:]))
	indent := strings.Repeat(" ", len(parent_key)+3)
	return strings.Join(ar, "\\\n"+indent)
}

func needs_heredoc(str string) bool {
	// if string has more than 3 newlines
	if strings.Count(str, "\n") > 3 {
		return true
	}
	return false
}

func output_heredoc(str string) string {
	indent := ""
	code := toUpper(fmt.Sprintf("__%x__", time.Now().Unix()))
	var heredoc []string
	heredoc = strings.Split(str, "\n")
	str = "<<" + code + "\n"
	for _, s := range heredoc {
		str += indent + s + "\n"
	}
	str += indent + code
	return str
}

func (o *Encoder) write_kv(depth int, key string, v interface{}) {
	key = setKeyCase(o.options, key)
	o.write(depth, fmt.Sprintf("%s = %v\n", key, v))
}

func (o *Encoder) write(depth int, s string) {
	indent := ""
	for i := depth; i > 1; i-- {
		indent += "  "
	}
	_, err := o.writer.Write([]byte(indent + s))
	if err != nil {
		o.appendErr("%s", err)
	}
}

func (o *Encoder) isOption(opt int) bool {
	return opt == opt&o.options
}

func quote(s string) string {
	if len(s) == 0 {
		return ""
	}
	q := strconv.QuoteToASCII(s)
	l := len(q)
	if q[1:l-1] != s {
		// return quoted string
		return q
	} else {
		// if we have leading or trailing space, return quoted string
		l := len(s)
		if s[0:1] == " " || s[l-1:] == " " {
			return q
		}
	}
	// just return original string
	return s
}

// Horked from unicode package
func toUpper(s string) string {
	z := []byte(s)
	for i := 0; i < len(z); i++ {
		z[i] = upper(z[i])
	}
	return string(z)
}

// Horked from unicode package
func upper(r byte) byte {
	if 'a' <= r && r <= 'z' {
		r -= 'a' - 'A'
	}
	return r
}

func isStructPtr(x interface{}) bool {
	v1 := reflect.ValueOf(x)
	if v1.Kind().String() == "ptr" {
		return v1.Elem().Kind().String() == "struct"
	}
	return false
}

func isTimeType(v interface{}) bool {
	return v == reflect.TypeOf(time.Time{})
}

func isDateOnly(t time.Time) bool {
	return !isTimeOffset(t) && t.Format(time_fmt) == "00:00:00"
}
func isTimeOnly(t time.Time) bool {
	return !isTimeOffset(t) && t.Format(date_fmt) == "0000-01-01"
}
func isUTCTime(t time.Time) bool {
	return isTimeOffset(t) && t.Format(date_fmt) == "0000-01-01"
}
func isUTCDate(t time.Time) bool {
	return isTimeOffset(t) && t.Format(date_fmt) != "0000-01-01"
}
func isDateTime(t time.Time) bool {
	return !isTimeOffset(t) && !isDateOnly(t) && !isTimeOnly(t)
}
func isTimeOffset(t time.Time) bool {
	return t.Format("-0700") != "+0000"
}
