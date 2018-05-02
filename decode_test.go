// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"fmt"
	"bytes"
	"time"
	"reflect"
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDecoder_Decode_strings(t *testing.T) {

	type c struct{ title, cfg, key, expected string }
	var tests []c

	tests = []c{
		c{"Plain String (colon assignment character)",
			`PlainString: They're just robots! It's okay to shoot them!`,
			"PlainString",
			"They're just robots! It's okay to shoot them!"},
		c{"Quoted String (equal assignment character)",
			`QuotedString = "    We need 10cc of concentrated dark matter or he'll die.    "`,
			"QuotedString",
			"    We need 10cc of concentrated dark matter or he'll die.    "},
		c{"Special Characters (no assignment character)",
			`SpecialChars = "What is my purpose?\nYou pass butter.\t\u263a"`,
			"SpecialChars",
			"What is my purpose?\nYou pass butter.\t\u263a"},
		c{"Heredoc Content",
			`Content = <<__A5A3231B__
<article>
  You know the worst part about inventing teleportation?
  Suddenly, you're able to travel the whole galaxy, and the
  first thing you learn is, you're the last guy to invent
  teleportation.
</article>
			__A5A3231B__`,
			"Content",
			"<article>\n" +
				"  You know the worst part about inventing teleportation?\n" +
				"  Suddenly, you're able to travel the whole galaxy, and the\n" +
				"  first thing you learn is, you're the last guy to invent\n" +
				"  teleportation.\n" +
				"</article>"},

		c{"MultiLine String without quotes",
			`MultiLine = There are pros and cons to every alternate timeline. Fun facts \
						 about this one: It has giant, telpathic spiders, eleven 9-11s, and \
						 the best ice cream in the multiverse!`,
			"MultiLine",
			"There are pros and cons to every alternate timeline. Fun facts about this one: " +
				"It has giant, telpathic spiders, eleven 9-11s, and the best ice cream in the multiverse!"},

		c{"MultiLine String with quotes",
			`MultiLine = "There are pros and cons to every alternate timeline. Fun facts "    \
						 "about this one: It has giant, telpathic spiders, eleven 9-11s, and "\
						 "the best ice cream in the multiverse!"`,
			"MultiLine",
			"There are pros and cons to every alternate timeline. Fun facts about this one: " +
				"It has giant, telpathic spiders, eleven 9-11s, and the best ice cream in the multiverse!"},

		c{"String Map",
			`StringMap {
				PlainString: They're just robots! It's okay to shoot them!
			 }`,
			"StringMap.PlainString",
			"They're just robots! It's okay to shoot them!"},
	}

	Convey("Decode a bunch of strings", t, func() {
		for i, test := range tests {
			fmt.Printf("\n  test %v %s", i, test.title)
			m := make(map[string]string)
			o := NewDecoder(m)
			err := o.DecodeString(test.cfg)
			So(err, ShouldBeNil)
			So(m[test.key], ShouldEqual, test.expected)
		}
	})

}


func TestDecoder_misc(t *testing.T) {

	// get more coverage
	_, err := floatFix("", 32)
	if err != nil {
		t.Fail()
	}

	type xt struct{ Key string }

//	Convey("extra field", t, func() {
//		var x struct {
//			Fld1 string
//		}
//		cfg := "foo = bar"
//		err := NewDecoder(&x).DecodeString(cfg)
//		So(err, ShouldNotBeNil)
//		println(err.Error())
//	})

}

func TestDecoder_unexported_field(t *testing.T) {

	Convey("Attempt to decode a private variable", t, func() {
		var x struct {
			Pub	string
			priv	string
		}
		cfg := `
			Pub  = Text
			priv = Text
		`
		err := Decode(&x, []byte(cfg))
		So( err, ShouldNotBeNil )
		So( err.Error(), ShouldEqual, "Extra field (priv) at line 3" )
		So( x.Pub,  ShouldEqual, "Text" )
		So( x.priv, ShouldEqual, "" )
	})

	Convey("Attempt to decode a private map", t, func() {
		var x struct {
			pm map[string]string
		}
		cfg := `
			pm {
				Key1 = Text
			}
		`
		err := Decode(&x, []byte(cfg))
		So( err, ShouldNotBeNil )
		So( err.Error(), ShouldEqual, "Extra field (pm.Key1) at line 3" )
	})

}


func TestDecoder_Decode_to_intmap(t *testing.T) {

	Convey("Decode separate strings to a map of integers", t, func() {

		cfg1 := `
		Int8    = 127
		Int16   = 32767`

		cfg2 := `
		Int32   = 2147483647
		Int64   = 9223372036854775807`

		m := make(map[string]int)
		o := NewDecoder(m)

		err := o.DecodeString(cfg1)
		So(err, ShouldBeNil)

		err = o.DecodeString(cfg2)
		So(err, ShouldBeNil)

		So(m["Int8"],  ShouldEqual, 127)
		So(m["Int16"], ShouldEqual, 32767)
		So(m["Int32"], ShouldEqual, 2147483647)
		So(m["Int64"], ShouldEqual, 9223372036854775807)

		for k, v := range m {
			fmt.Printf("%v  %v  \n", k, v)
		}

	})

}

func TestDecode_NumericTypes(t *testing.T) {

	Convey("Decode all numeric types", t, func() {
		cfg := `
		Int8    = 127
		Int16   = 32767
		Int32   = 2147483647
		Int     = 9223372036854775807
		Int64   = 9223372036854775807
		Uint8   = 255
		Uint16  = 65535
		Uint32  = 4294967295
		Uint    = 18446744073709551615
		Uint64  = 18446744073709551615
		Float32 = 3.40282355e+38
		Float64 = 1.7976931348623157e+308
		`
		var x numStruct
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldBeNil)
		So(CompareStructValues(x, testNumeric), ShouldBeTrue)
	})

	Convey("Decode single numbers to improve test coverage", t, func() {
		var x struct {
			Int1   int
			Float1 float32
		}
		cfg := `
			Int1 = 1
			Float1 = 1
		`
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldBeNil)
		So(x.Int1, ShouldEqual, 1)
	})

}

func TestNewDecoder_Decode_types(t *testing.T) {

	Convey("Decode from a stream", t, func() {
		cfg := []byte(`Int = 9223372036854775807`)
		var x numStruct
		br := bytes.NewReader(cfg)
		err := NewDecoder(&x).DecodeStream(br)
		So(err, ShouldBeNil)
		So(x.Int, ShouldEqual, 9223372036854775807)
	})

	Convey("Decode a byte slice", t, func() {
		cfg := []byte(`Int = 9223372036854775807`)
		var x numStruct
		err := NewDecoder(&x).DecodeBytes(cfg)
		So(err, ShouldBeNil)
		So(x.Int, ShouldEqual, 9223372036854775807)
	})

	Convey("Decode a string", t, func() {
		cfg := "Int = 9223372036854775807"
		var x numStruct
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldBeNil)
		So(x.Int, ShouldEqual, 9223372036854775807)
	})

}

func TestDecode_types(t *testing.T) {

	cfg := []byte("Int = 9223372036854775807")


	Convey("Decode a string", t, func() {
		var x numStruct
		err := Decode(&x, string(cfg))
		So(err, ShouldBeNil)
		So(x.Int, ShouldEqual, 9223372036854775807)
	})

	Convey("Decode a byte slice", t, func() {
		var x numStruct
		err := Decode(&x, cfg)
		So(err, ShouldBeNil)
		So(x.Int, ShouldEqual, 9223372036854775807)
	})

	Convey("Decode a stream", t, func() {
		var x numStruct
		err := Decode(&x, bytes.NewReader(cfg))
		So(err, ShouldBeNil)
		So(x.Int, ShouldEqual, 9223372036854775807)
	})

}

func TestDecode_Map_o_Structs(t *testing.T) {
	type stk struct {
		Str1 string
		Int1 int
	}
	type xMap struct {
		M map[string]stk
	}
	var x xMap
	Convey("Decode a map of structs", t, func() {
		cfg := []byte(`
		M = {
			Key1 = {
				Str1 = String1
				Int1 = 41
			}

			Key2 = {
				Str1 = String2
				Int1 = 42
			}
		}
		`)
		err := Decode(&x, cfg)
		So(err, ShouldBeNil)
		So(x.M["Key1"].Int1, ShouldEqual, 41)
		So(x.M["Key1"].Str1, ShouldEqual, "String1")
		So(x.M["Key2"].Int1, ShouldEqual, 42)
		So(x.M["Key2"].Str1, ShouldEqual, "String2")
	})
}

func TestDecode_force_panic(t *testing.T) {

	Convey("NewDecoder forced panic: Option not allowed", t, func() {
		var x struct{}
		fn := func() {
			_ = NewDecoder(&x, PARSE_LOWER_CASE)
		}
		So(fn, ShouldPanic)
	})

	Convey("NewDecoder forced panic: Expecting pointer", t, func() {
		var x struct{ Key1 [20]byte }
		fn := func() {
			_ = NewDecoder(x)
		}
		So(fn, ShouldPanic)
	})

	Convey("NewDecoder forced panic: Expecting pointer", t, func() {
		var m map[int]string
		fn := func() {
			_ = NewDecoder(m)
		}
		So(fn, ShouldPanic)
	})

}

func TestDecode_Force_NumericErrors(t *testing.T) {

	type xt struct{ Key string }

	Convey("Forced error: Invalid syntax", t, func() {
		var x struct {
			Float1 float64
		}
		cfg := "Float1=.K"
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Invalid abbreviation", t, func() {
		var x struct {
			Float1 float64
		}
		cfg := "Float1 = 3.1A"
		err := NewDecoder(&x).DecodeString(cfg)
		if err != nil {
			So(err.Error(), ShouldEqual, "Invalid numeric abbreviation at line 1")
		}
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Bad integer", t, func() {
		var x struct {
			Float1 float32
		}
		cfg := "Float1 = 3.40282355e+39"
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Int overflow in mapped struct", t, func() {
		type sx struct {
			Int1 int8
		}
		var x struct {
			Map1 map[string]sx
		}
		cfg := `
			Map1 {
				Key1 {
					Int1 = 256
				}
			}
		`
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Bad date", t, func() {
		var x struct {
			Key1 time.Time
		}
		cfg := "Key1 = 2017-01-1"
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Duplicate key", t, func() {
		var x xt
		cfg := `
			Key1=String1
			Key1=String2
			`
		err := NewDecoder(&x).DecodeString(cfg)
		if err != nil {
			So(err.Error(), ShouldEqual, "Duplicate key at line 3")
		}
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Slice", t, func() {
		var x struct{ Key1 []string }
		cfg := `
			Key1=String1
			`
		err := NewDecoder(&x).DecodeString(cfg)
		if err != nil {
			So(err.Error(), ShouldEqual, "Key1 type slice not allowed")
		}
		So(err, ShouldNotBeNil)
	})

	Convey("Forced error: Array", t, func() {
		var x struct{ Key1 [20]byte }
		cfg := `Key1=String1`
		err := NewDecoder(&x).DecodeString(cfg)
		if err != nil {
			So(err.Error(), ShouldEqual, "type array not allowed at line 1")
		}
		So(err, ShouldNotBeNil)
	})

}

func TestDecode_NumericGrouping(t *testing.T) {

	Convey("Decode Numeric Grouping", t, func() {
		cfg := `
		Int8    = 127
	    Int16   = 32,767
	    Int32   = 2,147,483,647
	    Int     = 9,223,372,036,854,775,807
	    Int64   = 9,223,372,036,854,775,807
	    Uint8   = 255
	    Uint16  = 65,535
	    Uint32  = 4,294,967,295
	    Uint    = 18,446,744,073,709,551,615
	    Uint64  = 18,446,744,073,709,551,615
	    Float32 = 3.402,823,55e+38
	    Float64 = 1.797,693,134,862,315,7e+308
		`
		var x numStruct
		err := NewDecoder(&x).DecodeString(cfg)

		So(err, ShouldBeNil)
		So(CompareStructValues(x, testNumeric), ShouldBeTrue)

	})
}

func TestDecode_NumbericOverflow(t *testing.T) {

	Convey("Force overflow of all numeric types", t, func() {
		cfgs := []string{
			`Int8    = 128`,
			`Int16   = 32768`,
			`Int32   = 9223372036854775808`,
			`Int     = 9223372036854775808`,
			`Int64   = 9223372036854775808`,
			`Uint8   = 256`,
			`Uint16  = 65536`,
			`Uint32  = 18446744073709551616`,
			`Uint    = 18446744073709551616`,
			`Uint64  = 18446744073709551616`,
			`Float32 = 3.40282355e+39`,
			`Float64 = 1.7976931348623157e+309`,
		}
		for _, cfg := range cfgs {
			var x numStruct
			Convey("Overflow: "+cfg, func() {
				err := NewDecoder(&x).DecodeString(cfg)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "")
			})
		}
	})

	Convey("Force overflow of a numeric abbreviation", t, func() {
		cfgs := []string{
			"Int32   = 92E",
			"Uint32  = 18E",
		}
		for _, cfg := range cfgs {
			var x numStruct
			Convey("Overflow: "+cfg, func() {
				err := NewDecoder(&x).DecodeString(cfg)
				So(err, ShouldNotBeNil)
			})
		}
	})

}

func TestDecode_Options(t *testing.T) {

	Convey("Decode snake case keys", t, func() {
		var x struct {
			SomeSnakeCaseKey   string
			Camel2Snake        string
			KeyWith_Underscore string
		}
		cfg := `
			some_snake_case_key 	String1
			camel_2_snake   		String2
			key_with_underscore		String3
		`
		err := Decode(&x, cfg, ALLOW_SNAKE_CASE)
		So(err, ShouldEqual, nil)
		So(x.SomeSnakeCaseKey, ShouldEqual, "String1")
		So(x.Camel2Snake, ShouldEqual, "String2")
		So(x.KeyWith_Underscore, ShouldEqual, "String3")
	})

	Convey("Decode lower case keys", t, func() {
		var x struct {
			SomeCamelCaseKey string
		}
		cfg := `
			somecamelcasekey 	String1
		`
		err := Decode(&x, cfg, IGNORE_CASE)
		So(err, ShouldEqual, nil)
		So(x.SomeCamelCaseKey, ShouldEqual, "String1")
	})

}

func TestDecode_NumericAbbreviations(t *testing.T) {

	Convey("Given int with abbreviation", t, func() {
		var x numAbbrevStruct
		cfg := `
		    Ki:     1K
		    Mi:     2,048M
		    Gi:     3G
		    Ti:     1T
		    Pi:     4P
		    Ei:     9E
			Kf:		2.2K
			Mf:		2.3M
		    Gf:     2.4G
		    Tf:     3.1T
		    Pf:     4.2P
		    Ef:     5.3E
		`
		err := NewDecoder(&x).DecodeString(cfg)
		So(err, ShouldEqual, nil)
		So(CompareStructValues(x, testAbrev), ShouldBeTrue)
	})

}

func TestDecode_ForceError_ExtraFields(t *testing.T) {
	var x struct{ Key2 int }
	Convey("Force error: Check for extra fields", t, func() {
		cfg := `
			Key1 = 41
			Key2 = 42
			Key3 = 43
			`
		o := NewDecoder(&x)
		err := o.DecodeString(cfg)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Extra field")
	})
}

func TestDecodeFile_errors(t *testing.T) {

	tempfile1 := createTempFile("GOTEST_CONFIG")
	tempfile2 := createTempFile("GOTEST_CONFIG")

	Convey("Force error: Supply empty filename", t, func() {
		var x testConfigX
		err := DecodeFile("", &x)
		So(err, ShouldNotBeNil)
	})

	Convey("Force error: Attempt to read non-existent file", t, func() {
		var x testConfigX
		err := DecodeFile("non existent file.conf", &x)
		So(err, ShouldNotBeNil)
	})

	Convey("Force error: Decode included file with errors", t, func() {
		var x numStruct

		// config with errors
		cfg1 := `
			Int8 = 128
			Int16 = "non numeric"
		`
		// write config to a file
		writeFile(tempfile1, []byte(cfg1))
		defer os.Remove(tempfile1)

		// write an include statement to a file
		writeFile(tempfile2, []byte("include "+tempfile1))
		defer os.Remove(tempfile2)

		err := NewDecoder(&x).DecodeFile(tempfile2)
		So(err, ShouldNotBeNil)

	})

}

func TestDecodeFile(t *testing.T) {

	tempfile1 := createTempFile("GOTEST_CONFIG")

	Convey("Decode example file", t, func() {
		var x testConfigX
		o := NewDecoder(&x)
		err := o.DecodeFile(example_conf_file)
		So(err, ShouldBeNil)

		b1, err := Encode(x)
		So(err, ShouldBeNil)

		b2, err := Encode(testConfig)
		So(err, ShouldBeNil)

		So(string(b1), ShouldEqual, string(b2))
	})
return

	Convey("Decode included file", t, func() {
		writeFile(tempfile1, []byte("include "+example_conf_file))
		defer os.Remove(tempfile1)

		var x testConfigX
		err := NewDecoder(&x).DecodeFile(tempfile1)
		So(err, ShouldBeNil)

		b1, err := Encode(x)
		So(err, ShouldBeNil)

		b2, err := Encode(testConfig)
		So(err, ShouldBeNil)

		So(string(b1), ShouldEqual, string(b2))
	})

}

func CompareStructValues(x, y interface{}) bool {
	v1 := reflect.ValueOf(x)
	if isStructPtr(x) {
		v1 = v1.Elem()
	}
	for i, n := 0, v1.NumField(); i < n; i++ {
		this_key := v1.Type().Field(i).Name
		y_val, ok := getStructValue(y, this_key)
		_ = y_val
		if ok {
			x_val := v1.Field(i)
			if x_val.Type().String() != y_val.Type().String() {
				fmt.Printf("Key: %s,  Types do not match:  %v != %v\n", this_key, x_val.Type().String(), y_val.Type().String())
				return false
			}
			if fmt.Sprintf("%v", x_val) != fmt.Sprintf("%v", y_val) {
				fmt.Printf("Key: %s,  Values do not match:  %v != %v\n", this_key, fmt.Sprintf("%v", x_val), fmt.Sprintf("%v", y_val))
				return false
			}
		}
	}
	return true
}
