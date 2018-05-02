// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

/*
Config provides encoding and decoding routines for configuration files. This
package supports most of the built-in datatypes, including string, int8-64,
uint8-64, float32-64, time.Time, struct, and string-keyed maps. Deeply nested
structs are supported as well as maps of structs. The data types not supported
are complex64/128, byte arrays, and slices.

This package also provides a Parse function which will allow any configuration
data to be parsed directly into a string map.

At this writing, struct tags are not supported. However, optional flags provide
a means to convert all fields to lower case or snake_case for encoding and
decoding.
*/
package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	// ALLOW_SNAKE_CASE will cause the decoder to interpret snake case fields in
	// the configuration file if the supplied struct is using Pascal case, eg.
	// crew_members == CrewMembers. Decode will attempt to find the actual struct
	// field before trying snake case.
	ALLOW_SNAKE_CASE = 1 << iota

	// IGNORE_CASE will cause the decoder to interpret lower case fields in
	// the configuration file, eg. crewmembers == CrewMembers. Decode will first
	// attempt to find the actual struct field before trying lower case.
	IGNORE_CASE

	// PARSE_LOWER_CASE will cause the parser to convert all keys to lower case.
	PARSE_LOWER_CASE

	// ENCODE_SNAKE_CASE will cause the encoder to convert all fields into
	// snake case, eg., DarkMatter == dark_matter.
	ENCODE_SNAKE_CASE

	// ENCODE_SNAKE_CASE will cause the encoder to convert all fields into
	// snake case, eg., DarkMatter == darkmatter.
	ENCODE_LOWER_CASE

	// ENCODE_ZERO_VALUES will cause zero values in the supplied struct to be encoded.
	ENCODE_ZERO_VALUES

	// OVERWRITE_FILE will cause the function EncodeToFile() to overwrite the
	// supplied filename if it already exists.
	OVERWRITE_FILE
)

// The Decoder converts the parsed data to the expected data type and assignes it to a struct.
type Decoder struct {
	reader   io.Reader
	lineno   int
	options  int
	fieldMap fMap
	v        interface{}
	parser   *Parser
	isMap    bool
	errs     []error
}


// NewDecoder accepts a pointer to a struct or a map and returns a new Decoder.
func NewDecoder(x interface{}, options ...int) *Decoder {
	o := &Decoder{}
	o.v = x
	switch {
	case reflect.TypeOf(x).Kind() == reflect.Map:
		if reflect.TypeOf(x).Key().Kind() != reflect.String {
			panic("Expecting map with string keys")
		}
		o.isMap = true
		return o
	case isStructPtr(x):
		break
	default:
		panic("Expecting pointer to a struct or a map")
	}
	if len(options) > 0 {
		if !o.allowedOption(options[0]) {
			panic("Option not allowed")
		}
		o.options = options[0]
	}
	return o
}

func (o *Decoder) allowedOption(option int) bool {
	return option == option&(ALLOW_SNAKE_CASE|ENCODE_SNAKE_CASE|IGNORE_CASE|ENCODE_LOWER_CASE)
}

// DecodeStream will accept an io.Reader
func (o *Decoder) DecodeStream(r io.Reader) error {
	o.parser = NewParser()
	o.reader = r
	return o.decode()
}

// DecodeBytes will accept a byteslice
func (o *Decoder) DecodeBytes(bs []byte) error {
	o.parser = NewParser()
	o.reader = bytes.NewReader(bs)
	return o.decode()
}

// DecodeString will accept a string
func (o *Decoder) DecodeString(s string) error {
	o.parser = NewParser()
	o.reader = strings.NewReader(s)
	return o.decode()
}

// Decode will accept a string, byte slice, or anything that implements an io.Reader
func Decode(x interface{}, src interface{}, options ...int) error {
	o := NewDecoder(x, options...)
	switch reflect.TypeOf(src).Kind() {
	case reflect.String:
		return o.DecodeString(src.(string))
	case reflect.Slice:
		return o.DecodeBytes(src.([]byte))
	default:
		// Hopefully, whatever you passed, implements an io.Reader.
		// If not, your code will panic right here.
		return o.DecodeStream(src.(io.Reader))
	}
}

// DecodeFile will decode the supplied filename
func (o *Decoder) DecodeFile(filename string) error {
	var err error
	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err = o.DecodeStream(fh); err != nil {
		return err
	}
	fh.Close()
	for _, f := range o.parser.include {
		if err := o.DecodeFile(f); err != nil {
			o.appendErr("%s\n", err.Error())
		}
	}
	return o.getErrs()
}

func (o *Decoder) appendErr(s string, v interface{}) {
	o.errs = append(o.errs, errors.New(fmt.Sprintf(s, v)))
}

func (o *Decoder) getErrs() error {
	var s string
	for _, e := range o.errs {
		s += e.Error() + "\n"
	}
	if s != "" {
		return errors.New(s)
	}
	return nil
}

// Decode the supplied source
func (o *Decoder) decode() error {
	var err error
	o.parser.reader = bufio.NewReader(o.reader)
	o.fieldMap, err = o.parser.parse()
	if err != nil {
		return err
	}
	if o.isMap {
		v1 := reflect.ValueOf(o.v)
		vt := v1.Type().Elem()
		for k, _ := range o.fieldMap {
			newValue := reflect.New(vt).Elem()
			if val, _, ok := o.getValue(k); ok {
				if err := setScalar(newValue, val); err == nil {
					v1.SetMapIndex(reflect.ValueOf(k), newValue)
				}
			}
		}
		return nil
	}
	err = o.traverseStruct(reflect.ValueOf(o.v), "")
	if err == nil {
		err = o.findExtraFields()
	}
	return err
}

// DecodeFile will decode the supplied file into the supplied
// struct. Decoder options are optional.
func DecodeFile(filename string, x interface{}, options ...int) error {
	return NewDecoder(x, options...).DecodeFile(filename)
}

func (o *Decoder) findExtraFields() error {
	var err error
	var msg string
	for k, v := range o.fieldMap {
		if !v.isDefined {
			if msg != "" {
				msg += "\n"
			}
			msg += fmt.Sprintf("Extra field (%s) at line %v", k, v.no)
		}
	}
	if msg != "" {
		err = errors.New(msg)
	}
	return err
}

func (o *Decoder) traverseStruct(v1 reflect.Value, parent_key string) error {
	switch v1.Kind() {
	case reflect.Slice:
		return newError(parent_key+" type slice not allowed", 0)
	case reflect.Struct:
		return o.iterateStructFields(v1, parent_key)
	case reflect.Map:
		return o.traverseMap(v1, parent_key)
	case reflect.Interface, reflect.Ptr:
		return o.traverseStruct(v1.Elem(), parent_key)
	default:
		if val, lineno, ok := o.getValue(parent_key); ok && v1.CanSet() {
			if err := setScalar(v1, val); err != nil {
				return newError(err.Error(),lineno)
			}
		}
	}
	return nil
}

func (o *Decoder) iterateStructFields(v1 reflect.Value, parent_key string) error {
	if isTimeType(v1.Type()) {
		if val, lineno, ok := o.getValue(parent_key); ok && v1.CanSet() {
			if err := set_time(v1, val); err != nil {
				return newError(err.Error(), lineno)
			}
		}
		return nil
	}
	for i, n := 0, v1.NumField(); i < n; i++ {
		this_key := v1.Type().Field(i).Name
		if !isPublic(this_key) {
			continue
		}
		if parent_key != "" {
			this_key = parent_key + "." + this_key
		}
		if err := o.traverseStruct(v1.Field(i), this_key); err != nil {
			return err
		}
	}
	return nil
}

func (o *Decoder) traverseMap(v1 reflect.Value, parent_key string) error {
	if v1.Type().Elem().Kind() != reflect.Struct {
		return o.traverseScalarMap(v1, parent_key)
	}
	if isTimeType(v1.Type().Elem()) {
		return o.traverseScalarMap(v1, parent_key)
	}
	v1.Set(reflect.MakeMap(v1.Type()))
	pkey := setKeyCase(o.options, parent_key)
	for mapkey, v := range o.fieldMap {
		v.kind = v1.Kind()
		if strings.Index(mapkey, pkey+".") == 0 {
			l := len(pkey) + 1

			if i := strings.Index(mapkey[l:], "."); i >= 0 {
				k := mapkey[l : l+i]
				key := mapkey[0 : l+i]
				newValue := reflect.New(v1.Type().Elem()).Elem()
				if err := o.traverseStruct(newValue, key); err != nil {
					return err
				}
				v1.SetMapIndex(reflect.ValueOf(k), newValue)
			}
		}
	}
	return nil
}

func (o *Decoder) traverseScalarMap(v1 reflect.Value, parent_key string) error {
	v1.Set(reflect.MakeMap(v1.Type()))
	pkey := setKeyCase(o.options, parent_key)
	for mapkey, v := range o.fieldMap {
		v.kind = v1.Kind()
		if strings.Index(mapkey, pkey+".") == 0 {
			k := mapkey[len(pkey)+1:]
			newValue := reflect.New(v1.Type().Elem()).Elem()
			if val, _, ok := o.getValue(mapkey); ok {
				if err := setScalar(newValue, val); err == nil {
					v1.SetMapIndex(reflect.ValueOf(k), newValue)
				}
			}
		}
	}
	return nil
}

func setKeyCase(option int, k string) string {
	if isOption(ALLOW_SNAKE_CASE, option) || isOption(ENCODE_SNAKE_CASE, option) {
		k = toSnakeCase(k)
	}
	if isOption(IGNORE_CASE, option) || isOption(ENCODE_LOWER_CASE, option) {
		k = toLower(k)
	}
	return k
}

func setScalar(v1 reflect.Value, val string) error {
	var err error
	switch v1.Kind() {
	case reflect.Struct:
		if isTimeType(v1.Type()) {
			err = set_time(v1, val)
		}
	case reflect.String:
		v1.SetString(val)
	case reflect.Bool:
		set_bool(v1, val)
	case reflect.Int8, reflect.Int16, reflect.Int32:
		err = set_int(v1, val)
	case reflect.Int64, reflect.Int:
		err = set_int64(v1, val)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		err = set_uint(v1, val)
	case reflect.Uint64, reflect.Uint:
		err = set_uint64(v1, val)
	case reflect.Float32, reflect.Float64:
		err = set_float(v1, val)
	default:
		err = errors.New(fmt.Sprintf("type %v not allowed", v1.Kind()))
	}
	return err
}

func set_time(v1 reflect.Value, val string) error {
	var tformat string
	switch len(val) {
	case 25:
		tformat = utc_date
	case 19:
		tformat = date_time
	case 14:
		tformat = utc_time
	case 10:
		tformat = date_fmt
	case 8:
		tformat = time_fmt
	default:
	}
	t, err := time.Parse(tformat, val)
	if err == nil {
		v1.Set(reflect.ValueOf(t))
	}
	return err
}

func set_bool(v1 reflect.Value, val string) {
	val = toLower(val)
	if val == "true" || val == "yes" || val == "on" || val == "1" {
		v1.SetBool(true)
	}
	if val == "false" || val == "no" || val == "off" || val == "0" {
		v1.SetBool(false)
	}
}

func set_int(v1 reflect.Value, val string) error {
	val = iFix(val)
	v, err := strconv.Atoi(val)
	if err == nil {
		if v1.OverflowInt(int64(v)) {
			return errors.New("Overflow")
		}
		v1.SetInt(int64(v))
	}
	return err
}

func set_int64(v1 reflect.Value, val string) error {
	v, err := strconv.ParseInt(iFix(val), 10, 64)
	if err == nil {
		v1.SetInt(int64(v))
	}
	return err
}

func set_uint(v1 reflect.Value, val string) error {
	val = iFix(val)
	v, err := strconv.Atoi(val)
	if err == nil {
		if v1.OverflowUint(uint64(v)) {
			return errors.New("Overflow")
		}
		v1.SetUint(uint64(v))
	}
	return err
}

func set_uint64(v1 reflect.Value, val string) error {
	v, err := strconv.ParseUint(iFix(val), 10, 64)
	if err == nil {
		v1.SetUint(uint64(v))
	}
	return err
}

func set_float(v1 reflect.Value, val string) error {
	var v float64
	var err error
	if v1.Kind() == reflect.Float32 {
		v, err = floatFix(val, 32)
	} else {
		v, err = floatFix(val, 64)
	}
	if err == nil {
		v1.SetFloat(v)
	}
	return err
}

func (o *Decoder) getValue(k string) (string, int, bool) {
	if vs, ok := o.fieldMap[k]; ok {
		vs.isDefined = true
		return vs.val, vs.no, true
	}
	if vs, ok := o.fieldMap[toSnakeCase(k)]; isOption(ALLOW_SNAKE_CASE, o.options) && ok {
		vs.isDefined = true
		return vs.val, vs.no, true
	}
	if vs, ok := o.fieldMap[toLower(k)]; isOption(IGNORE_CASE, o.options) && ok {
		vs.isDefined = true
		return vs.val, vs.no, true
	}
	return "", 0, false
}

func iFix(s string) string {
	if len(s) < 2 {
		return s
	}
	s = strings.Replace(s, ",", "", -1)  // remove commas
	n := len(s) - 1
	switch s[n] {
	case 'K':
		return s[:n] + "000"
	case 'M':
		return s[:n] + "000000"
	case 'G':
		return s[:n] + "000000000"
	case 'T':
		return s[:n] + "000000000000"
	case 'P':
		return s[:n] + "000000000000000"
	case 'E':
		return s[:n] + "000000000000000000"
	default:
		return s
	}
}

func floatFix(s string, b int) (float64, error) {
	n := len(s)
	switch {
	case n == 0:
		return 0, nil
	case n == 1:
		return strconv.ParseFloat(s, b)
	}
	s = strings.Replace(s, ",", "", -1)  // remove commas
	n = len(s) - 1
	c := s[n]
	if c >= '0' && c <= '9' {
		return strconv.ParseFloat(s, b)
	}
	v, err := strconv.ParseFloat(s[:n], b)
	if err != nil {
		return 0, err
	}
	switch c {
	case 'K':
		return v * 1e3, nil
	case 'M':
		return v * 1e6, nil
	case 'G':
		return v * 1e9, nil
	case 'T':
		return v * 1e12, nil
	case 'P':
		return v * 1e15, nil
	case 'E':
		return v * 1e18, nil
	default:
		return 0, errors.New("Invalid numeric abbreviation")
	}
}

// Convert a camel case key to snake case.
// Insert underscore at lower case to upper case boundary
// and at both sides of a number.
// Eg., SomeKey -> some_key, This2That -> this_2_that
func toSnakeCase(s string) string {
	var lastn, lastu, lastw bool
	var i int
	var bs string
	for _, c := range []byte(s) {
		i++
		n := isNumber(c)
		w := isLower(c)
		u := isUpper(c)
		if c == '_' {
			i = 0
		}
		if i > 1 && n != lastn {
			bs += "_"
		} else {
			if i > 1 && u != lastu && lastw {
				bs += "_"
				i = 0
			}
		}
		bs += string(lower(c))
		lastn = n
		lastu = u
		lastw = w
	}
	return bs
}

func isPublic(s string) bool {
	return isUpper(s[0])
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func isNumber(c byte) bool {
	return c >= '0' && c <= '9'
}

//func setCase__SAVE(opt int, k string) string {
//	if ALLOW_SNAKE_CASE == ALLOW_SNAKE_CASE&opt ||
//		ENCODE_SNAKE_CASE == ENCODE_SNAKE_CASE&opt {
//		k = toSnakeCase(k)
//	}
//	if IGNORE_CASE == IGNORE_CASE&opt ||
//		ENCODE_LOWER_CASE == ENCODE_LOWER_CASE&opt {
//		k = toLower(k)
//	}
//	return k
//}

func toLower(s string) string {
	z := []byte(s)
	for i := 0; i < len(z); i++ {
		z[i] = lower(z[i])
	}
	return string(z)
}

func lower(r byte) byte {
	if 'A' <= r && r <= 'Z' {
		r += 'a' - 'A'
	}
	return r
}

func newError(msg string, no int) error {
	if no > 0 {
		return errors.New(fmt.Sprintf("%s at line %d", msg, no))
	}
	return errors.New(msg)
}
