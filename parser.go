// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"bytes"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	multi_line_width = 80
	qt               = "\x22"
	lf               = "\n"
	comment        = "comment"
	open_brace     = "open_brace"
	close_brace    = "close_brace"
	keyval         = "keyval"
	multiline      = "multiline"
	multiline_cont = "multiline_cont"
	heredoc        = "heredoc"
	include        = "include"
	quoted         = "quoted"
	badkey         = "badkey"
	nested         = "~NESTED~"

	time_fmt  = "15:04:05"
	date_fmt  = "2006-01-02"
	utc_time  = "15:04:05 -0700"
	date_time = "2006-01-02 15:04:05"
	utc_date  = "2006-01-02 15:04:05 -0700"
)

// fMap contains key/value/line information for each field in source
type fMap map[string]*v

type v struct {
	val       string // String value of key/value from source
	no        int    // Line number of key/value from source
	isDefined bool   // Indicates that this field has been defined in the
	// given struct.  If this bool has not been set after
	// decode has completed, it will be considered extra.
	kind reflect.Kind //
}

// The Parser handles parsing input data from a reader.
type Parser struct {
	reader   *bufio.Reader
	lineno   int
	options  int
	errs     []error
	fieldMap fMap
	include  []string
	v        interface{}
}

// Type StringMap is the data type output by the Parse function.
type StringMap map[string]string

type matches struct {
	a []string
}

type rMap map[string]*regexp.Regexp

var compiledRegexp rMap

// Compile a few regular expressions
func init() {
	r := regexp.MustCompile
	compiledRegexp = rMap{
		comment:        r(`([^#]*)[#]`),
		open_brace:     r(`^([\w]+)\s*[=:\s]\s*{`),
		close_brace:    r(`^\s*}`),
		keyval:         r(`^\s*([\w\.]+)\s*[=:\s]\s*(.+)`), // allow all chars or just chars between quotes
		heredoc:        r(`^\s*([\w\.]+)\s*[=:\s]\s*<<([\w]+)`),
		multiline:      r(`^\s*([\w\.]+)\s*[=:\s]\s*(.*)\\$`),
		multiline_cont: r(`^\s*([^\\]*)\\$`),
		quoted:         r(`^"(.+)"\s*$`),
		include:        r(`^(?i)include +(\"?[^\"=]*)\"?$`),
		badkey:         r(`^\.|\.$|\.\.|^_$`), // match leading dot, trailing dot, adjacent dots, or a single underscore
	}
}

// NewParser returns a new Parser.
func NewParser(options ...int) *Parser {
	o := &Parser{}
	if len(options) > 0 {
		if !o.allowedOption(options[0]) {
			panic("Option not allowed")
		}
		o.options = options[0]
	}
	return o
}

func (o *Parser) allowedOption(option int) bool {
	return option == option&PARSE_LOWER_CASE
}

// Parse a string, a byte slice or an io.Reader to a string map.
func Parse(src interface{}, options ...int) (StringMap, error) {
	switch reflect.TypeOf(src).Kind() {
	case reflect.String:
		return NewParser(options...).ParseStream(strings.NewReader(src.(string)))
	case reflect.Slice:
		return NewParser(options...).ParseStream(bytes.NewReader(src.([]byte)))
	default:
		return NewParser(options...).ParseStream(src.(io.Reader))
	}
}

// Parse a file
func ParseFile(filename string, options ...int) (StringMap, error) {
	var err error
	f, err := os.Open(filename)
	if err != nil {
		return StringMap{}, err
	}
	defer f.Close()
	o := NewParser(options...)
	smap,_ := o.ParseStream(f)
	f.Close()
	for _, fname := range o.include {
		m,err := ParseFile(fname, options...)
		if err != nil {
			o.appendError("Errors in included file: "+fname+" (\n"+err.Error()+"\n)", 0)
		}
		for k,v := range m {
			smap[k] = v
		}
	}
	return smap, getErrors(o.errs)
}

// Parse a byte slice to a string map.
func (o *Parser) Parse(bs []byte) (StringMap, error) {
	return o.ParseStream(bytes.NewReader(bs))
}

// Parse a stream to a string map.
func (o *Parser) ParseStream(r io.Reader) (StringMap, error) {
	o.reader = bufio.NewReader(r)
	smap := make(StringMap)
	vmap, err := o.parse()
	for k, v := range vmap {
		if isOption(PARSE_LOWER_CASE, o.options) {
			k = toLower(k)
		}
		smap[k] = v.val
	}
	return smap, err
}

func (o *Parser) parse() (fMap, error) {
	vmap, _ := o.recursive_parse(0)
	if len(vmap) == 0 && len(o.include) == 0 {
		o.appendError("Nothing parsed", 0)
	}
	return vmap, getErrors(o.errs)
}

func (o *Parser) recursive_parse(depth int) (fMap, error) {
	var s string
	var err error
	m := matches{make([]string, 0, 0)}
	fieldMap := make(fMap)
	defer func() {
		// remove nested placeholders
		for key, vs := range fieldMap {
			if vs.val == nested {
				delete(fieldMap, key)
			}
		}
	}()
	for {
		s, err = o.nextLine()
		if err != nil {
			if err.Error() == "EOF" {
				err = nil
				if depth > 0 {
					return fieldMap, errors.New("Missing closing brace")
				}

			}
			break
		}
		switch {
		case findSubmatch(include, s, &m):
			o.include = append(o.include, m.a[1])

		case findSubmatch(open_brace, s, &m):
			key := m.a[1]
			lineno := o.lineno
			// recursive
			emap, err := o.recursive_parse(depth + 1)
			if err != nil {
				o.appendError(err.Error(), lineno)
				break
			}
			if exists(fieldMap, key) {
				o.appendError("Duplicate key", lineno)
				break
			} else {
				fieldMap[key] = &v{nested, lineno, false, 0}
			}
			for k, val := range emap {
				fieldMap[key+"."+k] = val
			}

		case findSubmatch(close_brace, s, &m):
			return fieldMap, nil

		case findSubmatch(heredoc, s, &m):
			key := m.a[1]
			code := m.a[2]
			val, err := o.readHereDoc(code)
			if err != nil {
				o.appendError(err.Error(), o.lineno)
				break
			}
			if exists(fieldMap, key) {
				o.appendError("Duplicate key", o.lineno)
				break
			}
			val, err = unquote(val)
			if err != nil {
				o.appendError(err.Error(), o.lineno)
				break
			}
			fieldMap[key] = &v{val, o.lineno, false, 0}

		case findSubmatch(multiline, s, &m):
			key := m.a[1]
			val := m.a[2]
			val = o.readMultiLine(val)
			if exists(fieldMap, key) {
				o.appendError("Duplicate key", o.lineno)
				break
			}
			val, err = unquote(val)
			if err != nil {
				o.appendError(err.Error(), o.lineno)
				break
			}
			fieldMap[key] = &v{val, o.lineno, false, 0}

		case findSubmatch(keyval, s, &m):
			key := m.a[1]
			val := m.a[2]
			if exists(fieldMap, key) {
				o.appendError("Duplicate key", o.lineno)
				break
			}
			if badKey(key) {
				o.appendError("Invalid key", o.lineno)
				break
			}
			val, err = unquote(val)
			if err != nil {
				o.appendError(err.Error(), o.lineno)
				break
			}
			fieldMap[key] = &v{val, o.lineno, false, 0}

		default:
			o.appendError("Invalid data", o.lineno)

		}
	}
	return fieldMap, nil
}

func badKey(k string) bool {
	m := matches{make([]string, 0, 0)}
	return findSubmatch(badkey, k, &m)
}

func findSubmatch(key, s string, m *matches) bool {
	m.a = compiledRegexp[key].FindStringSubmatch(s)
	return m.a != nil
}

func (o *Parser) readMultiLine(content string) string {
	m := matches{make([]string, 0, 0)}
	if findSubmatch(quoted, content, &m) {
		content = m.a[1]
	}
	for {
		s, err := o.nextLine()
		if err != nil {
			o.appendError("EOF encountered before multiline termination",o.lineno)
			break
		}
		if !findSubmatch(multiline_cont, s, &m) {
			if findSubmatch(quoted, s, &m) {
				s = m.a[1]
			}
			content += s
			break
		}
		s = m.a[1]
		if findSubmatch(quoted, s, &m) {
			s = m.a[1]
		}
		content += s
	}
	return content
}

func (o *Parser) nextLine() (s string, err error) {
	m := matches{make([]string, 0, 0)}
	for {
		b, err := o.reader.ReadBytes('\n')
		s = string(b)
		if err != nil {
			if err.Error() == "EOF" && s != "" {
				// we still have data. keep going
				err = nil
			} else {
				return "", err
			}
		}
		o.lineno++
		if findSubmatch(comment, s, &m) {
			s = m.a[1]
		}
		s = trim(s)
		if s != "" {
			break
		}
	}
	return s, err
}

func (o *Parser) readHereDoc(code string) (string, error) {
	var content string
	var s string
	var isCode bool
	for {
		b, e := o.reader.ReadBytes('\n')
		if e != nil {
			if len(b) == 0 {
				break
			}
		}
		s = string(b)
		o.lineno++
		if code == trim(s) {
			isCode = true
			break
		}
		s = rtrim(s)
		if content != "" {
			content += "\n"
		}
		content += s
	}
	var err error
	if !isCode {
		err = errors.New("No terminating heredoc code")
	}
	return content, err
}

// Includes will return a list of file names that have been included in the
// source configuration file.
func (o *Parser) Includes() []string {
	return o.include
}

func unquote(s string) (string, error) {
	l := len(s)
	if l == 0 {
		return "", nil
	}
	// remove boundary quotes
	if s[0:1] == qt && s[l-1:l] == qt {
		s = s[1 : l-1]
	}
	s = strings.Replace(s, lf, `\n`, -1)
	// temporarily replace embedded quotes
	s = strings.Replace(s, qt, `\x22`, -1)
	t, err := strconv.Unquote(qt + s + qt)
	if err != nil {
		err = errors.New(err.Error() + ": Unquote(" + s + ")")
	} else {
		s = t
	}
	// put the embedded quotes back the way they were
	s = strings.Replace(s, `\x22`, qt, -1)
	return s, err
}

// Trim leading and trailing white space
func trim(s string) string {
	var n int
	for n = len(s) - 1; n >= 0; n-- {
		if !isWhiteSp(s[n]) {
			break
		}
	}
	s = s[:n+1]
	for n = 0; n < len(s); n++ {
		if !isWhiteSp(s[n]) {
			break
		}
	}
	return s[n:]
}

// Trim trailing white space
func rtrim(s string) string {
	var n int
	for n = len(s) - 1; n >= 0; n-- {
		if !isWhiteSp(s[n]) {
			break
		}
	}
	return s[:n+1]
}

// Return true if may key exists
func exists(m fMap, key string) bool {
	_, ok := m[key]
	return ok
}

// Return true if the character is white space
func isWhiteSp(c byte) bool {
	if (c >= 9 && c <= 13) || c == 32 {
		return true
	}
	return false
}

func (o *Parser) appendError(msg string, no int) {
	if no > 0 {
		msg = fmt.Sprintf("%s at line %d", msg, no)
	}
	o.errs = append(o.errs, errors.New(msg))
}

func getErrors( errs []error ) error {
	var s string
	if len(errs) == 0 {
		return nil
	}
	for _, e := range errs {
		s += e.Error() + "\n"
	}
	s = strings.TrimRight(s, "\n")
	return errors.New(s)
}

func isOption(option, options int) bool {
	return option == option&options
}
