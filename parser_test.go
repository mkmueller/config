// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
//	"log"
//	"fmt"
//	"bufio"
	"strings"
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParse_function(t *testing.T) {

	cfg := `
		NoAssigmentOperator Are you far away, or are you inside something?
		ColonAssigmentOperator: Is this a camera? Is everything a camera?
		PlainString     = I don't do magic, I do science. One takes brains, the other takes dark eye liner.  #'
		SpecialChars    = \tThe reason anyone would do this,\n\tif they could, which they can't,\n\twould be because they could, which they can't.\u1f636\n
		QuotedString    = "  Did you do this on purpose to get out of family counseling?  "
		EmbeddedQuotes  = I assure you, I would never "find a way" to "get out of" family therapy.
	`

	tests := make(map[string]string)
	tests = map[string]string{
		"NoAssigmentOperator":    "Are you far away, or are you inside something?",
		"ColonAssigmentOperator": "Is this a camera? Is everything a camera?",
		"PlainString":            "I don't do magic, I do science. One takes brains, the other takes dark eye liner.",
		"SpecialChars":           "\tThe reason anyone would do this,\n\tif they could, which they can't,\n\twould be because they could, which they can't.\u1f636\n",
		"QuotedString":           "  Did you do this on purpose to get out of family counseling?  ",
		"EmbeddedQuotes":         `I assure you, I would never "find a way" to "get out of" family therapy.`,
	}

	Convey("Parse file with included file, lower case", t, func() {

		tempfile1 := createTempFile("GOTEST_CONFIG")
		tempfile2 := createTempFile("GOTEST_CONFIG")
		writeFile(tempfile1, []byte(cfg))
		writeFile(tempfile2, []byte("include " + tempfile1))
		defer os.Remove(tempfile1)
		defer os.Remove(tempfile2)

		m, err := ParseFile(tempfile1, PARSE_LOWER_CASE)
		So(err, ShouldBeNil)
		for _, key := range []string{"PlainString", "SpecialChars", "QuotedString", "EmbeddedQuotes"} {
			lkey := strings.ToLower(key)
			So(m[lkey], ShouldEqual, tests[key])
		}

		m, err = ParseFile(tempfile2, PARSE_LOWER_CASE)
		So(err, ShouldBeNil)
		for _, key := range []string{"PlainString", "SpecialChars", "QuotedString", "EmbeddedQuotes"} {
			lkey := strings.ToLower(key)
			So(m[lkey], ShouldEqual, tests[key])
		}

	})

	Convey("Parse from a string", t, func() {
		m, err := Parse(cfg)
		So(err, ShouldBeNil)
		for _, key := range []string{"PlainString", "SpecialChars", "QuotedString", "EmbeddedQuotes"} {
			So(m[key], ShouldEqual, tests[key])
		}
	})

	Convey("Parse from a byte slice", t, func() {
		m, err := Parse([]byte(cfg))
		So(err, ShouldBeNil)
		for _, key := range []string{"PlainString", "SpecialChars", "QuotedString", "EmbeddedQuotes"} {
			So(m[key], ShouldEqual, tests[key])
		}
	})

	Convey("Parse from a stream", t, func() {
		m, err := Parse(strings.NewReader(cfg))
		So(err, ShouldBeNil)
		for _, key := range []string{"PlainString", "SpecialChars", "QuotedString", "EmbeddedQuotes"} {
			So(m[key], ShouldEqual, tests[key])
		}
	})

	Convey("Parse lower case", t, func() {
		m, err := Parse(cfg, PARSE_LOWER_CASE)
		So(err, ShouldBeNil)
		for _, key := range []string{"PlainString", "SpecialChars", "QuotedString", "EmbeddedQuotes"} {
			lkey := strings.ToLower(key)
			So(m[lkey], ShouldEqual, tests[key])
		}
	})




}

func TestParser_Includes(t *testing.T) {

	cfg := `include myconfig.conf
			include /path/myconfig.conf`

	Convey("Parse bytes to get include lines", t, func() {
		p := NewParser()
		_, err := p.Parse([]byte(cfg))
		So(err, ShouldBeNil)
		So(len(p.Includes()), ShouldEqual, 2)
		So(p.Includes()[0], ShouldEqual, "myconfig.conf")
		So(p.Includes()[1], ShouldEqual, "/path/myconfig.conf")
	})

}

func TestParser_force_errors(t *testing.T) {

	type c struct{ cfg, errmsg string }
	var tests []c

	Convey("Forced errors with nothing parsed", t, func() {

		tests = []c{
			c{"Hdoc = <<_END", "No terminating heredoc code at line 1"},
			c{`Key1 = "foo\"`, "invalid syntax: Unquote(foo\\) at line 1"}, //"
			c{"SomeKey", "Invalid data at line 1"},
			c{"SomeKey=", "Invalid data at line 1"},
			c{"= Some String", "Invalid data at line 1"},
			c{"_ = Some string", "Invalid key at line 1"},
			c{"Key1..Key2 = Some string", "Invalid key at line 1"},
			c{"Key1. = Some string", "Invalid key at line 1"},
			c{".Key1 = Some string", "Invalid key at line 1"},
			c{".Key1 = 3\nKey2. = 4", "Invalid key at line 1\nInvalid key at line 2"},
			c{"Key1={Key=2\n", "Missing closing brace at line 1"},
		}

		for _, test := range tests {
			m, err := Parse([]byte(test.cfg))
			_ = m
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, test.errmsg+"\nNothing parsed")
		}

	})

	Convey("Forced errors", t, func() {

		tests = []c{
			c{"Key1=1\nKey1=2\n", "Duplicate key at line 2"},
			c{"Key1={\nKey2=2\n}\nKey1={\nKey2=2\n}\n", "Duplicate key at line 4"},
			c{` Hdoc = <<_END
				    Foo bar
				    _END
				Hdoc = <<_END
				    Foo bar
				    _END
				`, "Duplicate key at line 6"},
			c{`
Hdoc1 = <<_END
Foo bar
_END
Hdoc2 = <<_END
Foo bar \u00
_END
				`, "invalid syntax: Unquote(Foo bar \\u00) at line 7"},

			c{` Mline = Foo \
				Bar
				Mline = Foo \
				Bar
				`, "Duplicate key at line 4"},
			c{` Mline1 = Foo \
				Bar
				Mline2 = Foo \
				Bar \u00
				`, "invalid syntax: Unquote(Foo Bar \\u00) at line 4"},
			c{`Mline = string \`,
				"EOF encountered before multiline termination at line 1"},
			c{`Mline = \`,
				"EOF encountered before multiline termination at line 1"},
		}

		for _, test := range tests {
			_, err := Parse([]byte(test.cfg))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, test.errmsg)
		}

	})

}


func TestParser_force_panic(t *testing.T) {

	Convey("Create new parser with bad option", t, func() {
		fn := func(){
			_ = NewParser(IGNORE_CASE)
		}

		So( fn, ShouldPanic )
	})

}

func TestParseFile_errors(t *testing.T) {

	tempfile1 := createTempFile("GOTEST_CONFIG")
	tempfile2 := createTempFile("GOTEST_CONFIG")
	_ = tempfile2

//	Convey("Force errors: Parse file (with included file) with a bunch of errors", t, func() {
//
//		cfg := `MultiLine = There are pros and cons to every alternate timeline. Fun facts \
//		about this one: It has giant, telpathic spiders, eleven 9-11s, and \
//            	the best ice cream in the multiverse!\`
//
//		o := &Parser{}
//		o.reader = bufio.NewReader(strings.NewReader(cfg))
//		s,_ := o.nextLine()
//
//		// drop the stream (like deleting the file mid stream)
////		o.reader = bufio.NewReader(strings.NewReader("\\"))
//		s = o.readMultiLine(s)
//
//		println(s)
//
//	})
//return


	// config with errors
	cfg := `
		SomeKey                     # Key without assignment
		SomeKey=                    # Assignment without value
		= Some String               # Assignment without key
		_ = Some String             # Invalid key
		Key1..Key2 = Some String    # Invalid key
		Key1. = Some String         # Invalid key
		.Key1 = Some String         # Invalid key
		Key1={Key=2                 # Missing closing brace
	`

	Convey("Force error: Supply empty filename", t, func() {
		_,err := ParseFile("")
		So(err, ShouldNotBeNil)
	})

	Convey("Force error: Attempt to read non-existent file", t, func() {
		_,err := ParseFile("non existent file.conf")
		So(err, ShouldNotBeNil)
	})

	Convey("Force errors: Parse file with a bunch of errors", t, func() {

		// write config to a file
		writeFile(tempfile1, []byte(cfg))
		defer os.Remove(tempfile1)

		_,err := ParseFile(tempfile1)
		So(err, ShouldNotBeNil)

	})

	Convey("Force errors: Parse file (with included file) with a bunch of errors", t, func() {

		// write config to a file
		writeFile(tempfile1, []byte(cfg))
		defer os.Remove(tempfile1)

		writeFile(tempfile2, []byte("foo\nbar=\ninclude "+tempfile1))
		defer os.Remove(tempfile2)

		_,err := ParseFile(tempfile2)
		So(err, ShouldNotBeNil)

	})


}
