// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"fmt"
	"log"
	"time"
	"bytes"
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewEncoder(t *testing.T) {

	Convey("Encode Struct", t, func() {
		x := struct {
			MyPi float64
		}{3.14159265359}
		cfg := "MyPi = 3.14159265359\n"
		var buf bytes.Buffer
		o := NewEncoder(x)
		err := o.ToStream(&buf)
		So(err, ShouldBeNil)
		So(string(buf.Bytes()), ShouldEqual, cfg)
	})

	Convey("Encode a Struct Pointer", t, func() {
		x := struct {
			MyPi float64
		}{3.14159265359}
		cfg := "MyPi = 3.14159265359\n"
		var buf bytes.Buffer
		o := NewEncoder(&x)
		err := o.ToStream(&buf)
		So(err, ShouldBeNil)
		So(string(buf.Bytes()), ShouldEqual, cfg)
	})

	Convey("Force panic: pointer to a map", t, func() {
		m := make(map[string]string)
		m["Key1"] = "String1"
		fn := func() {
			o := NewEncoder(&m)
			_ = o
		}
		So(fn, ShouldPanic)
	})

	Convey("Force panic: wrong type", t, func() {
		s := "String1"
		fn := func() {
			o := NewEncoder(s)
			_ = o
		}
		So(fn, ShouldPanic)
	})

	Convey("Force panic: option not allowed", t, func() {
		x := struct {
			MyPi float64
		}{3.14159265359}
		fn := func() {
			o := NewEncoder(x, PARSE_LOWER_CASE)
			_ = o
		}
		So(fn, ShouldPanic)
	})

}

func TestEncode_EmbeddedStruct(t *testing.T) {

	Convey("Encode Simple Struct", t, func() {
		x := struct {
			Simple simpleStruct
		}{testSimple}
		cfg := "Simple = {\n  S = String1\n  I = 41\n}\n"
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

}

func TestEncode_Strings(t *testing.T) {

	Convey("Break a string at the right spot (for extra coverage)", t, func() {

		type xs struct {
			S		string
			MultiLine1	string
		}
		x := xs{
			"aa",
			//"aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa  aa",
			"We need to break this really long string at just the right spot (for extra coverage)",
		}
		Expected := `S = aa
MultiLine1 = We need to break this really long string at just the right spot (for\
             " extra coverage)"
`

//		var b1 []byte
//		o := NewEncoder(&x)
//		err := o.ToBytes(&b1)

		b1,err := Encode(&x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, Expected)

	})
return


	x := strStruct{
		PlainString:  testString.PlainString,
		QuotedString: testString.QuotedString,
		SpecialChars: testString.SpecialChars,
		MultiLine:    testString.MultiLine,
	}
	y := strStruct{
		Content: testString.Content,
	}

	cfg1 := `PlainString = They're just robots! It's okay to shoot them!
QuotedString = "    We need 10cc of concentrated dark matter or he'll die.    "
SpecialChars = "What is my purpose?\nYou pass butter.\t\u263a"
MultiLine = There are pros and cons to every alternate timeline. Fun facts about\
            " this one: It has giant, telpathic spiders, eleven 9-11s, and the best"\
            " ice cream in the multiverse!"
`
	cfg2 := `<article>
    You know the worst part about inventing teleportation?
    Suddenly, you're able to travel the whole galaxy, and the
    first thing you learn is, you're the last guy to invent
    teleportation.
</article>`

	m := matches{make([]string, 0, 0)}

	Convey("Encode Strings", t, func() {
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg1)
		b2, err := Encode(y)
		So(err, ShouldBeNil)
		if findSubmatch(heredoc, string(b2), &m) {
			key := m.a[1]
			code := m.a[2]
			b3 := fmt.Sprintf("%s = <<%s\n%s\n%s\n", key, code, cfg2, code)
			So(string(b2), ShouldEqual, b3)
		} else {
			t.Fail()
		}
	})


}

func TestEncode_Numeric_Values(t *testing.T) {

	Convey("Encode Numeric Values", t, func() {
		cfg := `Int8 = 127
Int16 = 32767
Int32 = 2147483647
Int = 9223372036854775807
Int64 = 9223372036854775807
Uint8 = 255
Uint16 = 65535
Uint32 = 4294967295
Uint = 18446744073709551615
Uint64 = 18446744073709551615
Float32 = 3.4028235e+38
Float64 = 1.7976931348623157e+308
`
		b1, err := Encode(testNumeric)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

}

func TestEncode_ForceErrors(t *testing.T) {

	var xStruct struct {
		Cplx complex128
	}

	Convey("Attempt to encode complex number", t, func() {
		_, err := Encode(xStruct)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "Cannot encode type (complex128)")
	})

	Convey("Force a write error", t, func() {
		tempfile := createTempFile("GOTEST_CONFIG")
		fh, err := os.Create(tempfile)
		if err != nil {
			log.Fatal(err.Error())
		}
		o := &Encoder{}
		o.writer = fh
		o.write(0, "write this\n")
		fh.Close()
		o.write(0, "write to closed file")
		err = getErrors(o.errs)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "write "+tempfile+": file already closed")
	})

}

func TestEncode_lowercase_fields(t *testing.T) {

	x := struct {
		UPPERCASEFIELD string
		MixedCaseField string
	}{"He's just a pickle", "He's a monster"}

	Convey("Encode Lower Case Fields", t, func() {
		cfg := "uppercasefield = He's just a pickle\n" +
			"mixedcasefield = He's a monster\n"
		b1, err := Encode(x, ENCODE_LOWER_CASE)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

	Convey("Encode Snake Case Fields", t, func() {
		cfg := "uppercasefield = He's just a pickle\n" +
			"mixed_case_field = He's a monster\n"
		b1, err := Encode(x, ENCODE_SNAKE_CASE)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

}

func TestEncode_Boolean_Values(t *testing.T) {

	Convey("Encode Boolean Values", t, func() {

		x := struct {
			Key1 bool
			Key2 bool
		}{true, false}

		cfg := "Key1 = True\nKey2 = False\n"

		b1, err := Encode(x, ENCODE_ZERO_VALUES)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)

	})

}

func TestEncode_ZeroValues(t *testing.T) {

	Convey("Encode Zero Values", t, func() {
		var x struct {
			Key1 bool
			Key2 string
			Key3 int
			Key4 time.Time
		}
		cfg := "Key1 = False\nKey2 = \"\"\nKey3 = 0\nKey4 = 0001-01-01\n"
		b1, err := Encode(x, ENCODE_ZERO_VALUES)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

}

func TestEncode_Time_Values(t *testing.T) {

	Convey("Encode Time Values", t, func() {

		cfg := `OffsetDateTime = 2017-12-25 08:10:00 -0800
DateTime = 2017-12-25 08:10:00
DateOnly = 2017-12-25
TimeOnly = 08:10:00
OffsetTime = 08:10:00 -0800
`
		b1, err := Encode(testTime)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)

	})

}

func TestEncode_Maps(t *testing.T) {

	Convey("Encode a map of floats", t, func() {
		x := make(map[string]float64)
		x["MyPi"] = 3.14159265359
		cfg := "MyPi = 3.14159265359\n"
		var buf bytes.Buffer
		o := NewEncoder(x)
		err := o.ToStream(&buf)
		So(err, ShouldBeNil)
		So(string(buf.Bytes()), ShouldEqual, cfg)
	})

	Convey("Encode A Map of String Values", t, func() {
		var x struct {
			M1 stringMap
		}
		x.M1 = stringMap{
			"Key1": "Jaguar was an animal.",
			"Key2": "You're an intelligent pickle.",
		}
		cfg := "M1 = {\n" +
			"  Key1 = Jaguar was an animal.\n" +
			"  Key2 = You're an intelligent pickle.\n" +
			"}\n"

		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

	Convey("Encode A Map of Integer Values", t, func() {
		x := struct {
			M1 intMap
		}{testIntMap}
		cfg := "M1 = {\n" +
			"  Key1 = 2147483647\n" +
			"  Key2 = -2147483648\n}\n"
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

	Convey("Encode A Map of Structs", t, func() {
		x := struct {
			St1 structMap
		}{testStructMap}
		cfg := `St1 = {
  Key1 = {
    S = String1
    I = 41
  }
  Key2 = {
    S = String2
    I = 42
  }
}
`
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

	Convey("Encode A Map of Structs with zero values. Zero values not encoded by default.", t, func() {
		type st struct {
			I	int
			S	string
		}
		type sm map[string]st
		x := struct {
			St1 	sm
		}{
			St1:	sm{
				"Key1": st{0,  "String1"},
				"Key2": st{42, ""},
			},
		}
		cfg := `St1 = {
  Key1 = {
    S = String1
  }
  Key2 = {
    I = 42
  }
}
`
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

	Convey("Encode A Map of Structs with zero values. Zero values are encoded with option.", t, func() {
		type st struct {
			I	int
			S	string
		}
		type sm map[string]st
		x := struct {
			St1 	sm
		}{
			St1:	sm{
				"Key1": st{0,  "String1"},
				"Key2": st{42, ""},
			},
		}
		cfg := `St1 = {
  Key1 = {
    I = 0
    S = String1
  }
  Key2 = {
    I = 42
    S = ""
  }
}
`
		b1, err := Encode(x,ENCODE_ZERO_VALUES)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})

	Convey("Encode A Map of Time Values", t, func() {
		x := struct {
			Map1 timeMap
		}{testTimeMap}
		cfg := `Map1 = {
  Key1 = 2017-12-25 08:10:00 -0800
  Key2 = 2017-12-25 08:10:00
  Key3 = 2017-12-25
  Key4 = 08:10:00
  Key5 = 08:10:00 -0800
}
`
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})
}

func TestEncode_Nested_Structs(t *testing.T) {

	Convey("Encode Nested Struct With Private Fields", t, func() {
		type ys struct {
			Pub    string
			priv   string
		}
		type xs struct {
			S1	ys
		}
		x := xs{
			S1: ys{
				Pub:  "Text1",
				priv: "Text2",
			},
		}
Expected := `S1 = {
  Pub = Text1
}
`
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, Expected)
	})

	Convey("Encode Nested Structs", t, func() {

		x := struct {
			Nested nestedStruct
		}{testNested}

		cfg := `Nested = {
  Level1 = {
    Level2 = {
      Level3 = {
        S = String1
        I = 41
      }
    }
  }
}
`
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)

	})

	Convey("Encode Nested Empty Struct", t, func() {
		var x struct {
			Nested struct{}
		}
		cfg := ""
		b1, err := Encode(x)
		So(err, ShouldBeNil)
		So(string(b1), ShouldEqual, cfg)
	})


}

func TestEncodeToFile(t *testing.T) {

	tempfile1 := createTempFile("GOTEST_CONFIG1")
	tempfile2 := createTempFile("GOTEST_CONFIG2")
	defer os.Remove(tempfile1)
	defer os.Remove(tempfile2)

	Convey("Attempt to write to file that already exists", t, func() {
		var x testConfigX
		o := NewEncoder(x)
		err := o.ToFile(tempfile1)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "file already exists")
	})

	Convey("Attempt to write empty config to a file", t, func() {
		os.Remove(tempfile1)
		var x testConfigX
		o := NewEncoder(x)
		err := o.ToFile(tempfile1)
		So(err, ShouldBeNil)
		So(fileExists(tempfile1), ShouldBeFalse)
	})

	Convey("Attempt to overwrite a directory", t, func() {
		err := os.Mkdir(tempfile1, 0700)
		So(err, ShouldBeNil)
		var x testConfigX
		err = EncodeToFile(x, tempfile1, OVERWRITE_FILE)
		So(err, ShouldNotBeNil)
		os.Remove(tempfile1)
	})

	Convey("Overwrite a file", t, func() {
		x := testConfigX{PlainString: "They're just robots! It's okay to shoot them!"}
		err := EncodeToFile(x, tempfile2, OVERWRITE_FILE)
		So(err, ShouldBeNil)
		So(fileExists(tempfile2), ShouldBeTrue)
		os.Remove(tempfile2)
	})

	Convey("Write config to a file", t, func() {
		os.Remove(tempfile2)
		err := EncodeToFile(testConfig, tempfile2)
		So(err, ShouldBeNil)
		So(fileExists(tempfile2), ShouldBeTrue)

		Convey("Read the file we just wrote and compare the values to the original", func() {
			var x testConfigX
			err := DecodeFile(tempfile2, &x)
			So(err, ShouldBeNil)
			b1, err := Encode(testConfig)
			b2, err := Encode(x)
			So(string(b1), ShouldEqual, string(b2))
		})

	})

	Convey("Force error: Supply empty filename", t, func() {
		var x testConfigX
		err := EncodeToFile(x, "")
		So(err, ShouldNotBeNil)
	})

}
