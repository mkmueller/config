// Copyright (c) 2018 Mark K Mueller <github.com/mkmueller>
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"io/ioutil"
	"strings"
	"log"
	"math"
	"os"
	"reflect"
	"time"
)

const example_conf_file = "example.conf"
const TEMP_DIR = "/tmp"

type simpleStruct struct {
	S string
	I int
}

var testSimple simpleStruct

type nestedStruct struct {
	Level1 struct {
		Level2 struct {
			Level3 simpleStruct
		}
	}
}

var testNested nestedStruct

type strStruct struct {
	EmptyString  string
	PlainString  string
	QuotedString string
	SpecialChars string
	MultiLine    string
	Content      string
}

var testString strStruct

type numStruct struct {
	ZeroInt   int
	Int8      int8
	Int16     int16
	Int32     int32
	Int       int
	Int64     int64
	ZeroUint  uint
	Uint8     uint8
	Uint16    uint16
	Uint32    uint32
	Uint      uint
	Uint64    uint64
	ZeroFloat float32
	Float32   float32
	Float64   float64
}

var testNumeric numStruct

type numAbbrevStruct struct {
	Ki int
	Mi int
	Gi int
	Ti int64
	Pi int64
	Ei int64
	Kf float32
	Mf float32
	Gf float32
	Tf float64
	Pf float64
	Ef float64
}

var testAbrev numAbbrevStruct

type boolStruct struct {
	Bool1 bool
	Bool2 bool
	Bool3 bool
	Bool4 bool
	Bool5 bool
	Bool6 bool
	Bool7 bool
	Bool8 bool
}

var testBool boolStruct

type timeStruct struct {
	OffsetDateTime time.Time
	DateTime       time.Time
	DateOnly       time.Time
	TimeOnly       time.Time
	OffsetTime     time.Time
}

var testTime timeStruct

type stringMap map[string]string
type intMap map[string]int
type structMap map[string]simpleStruct
type timeMap map[string]time.Time

var testStringMap stringMap
var testIntMap intMap
var testStructMap structMap
var testTimeMap timeMap

type testConfigX struct {
	PlainString  string
	QuotedString string
	SpecialChars string
	MultiLine    string
	Content      string
	Numeric      numStruct
	Abreviations numAbbrevStruct
	Bools        boolStruct
	Times        timeStruct
	StringMap    stringMap
	IntegerMap   intMap
	StructMap    structMap
	TimeMap      timeMap
	Nested       nestedStruct
}

var testConfig testConfigX

func init() {

	testSimple = simpleStruct{"String1", 41}
	testNested.Level1.Level2.Level3 = testSimple

	testString = strStruct{
		PlainString:  "They're just robots! It's okay to shoot them!",
		QuotedString: "    We need 10cc of concentrated dark matter or he'll die.    ",
		SpecialChars: "What is my purpose?\nYou pass butter.\t\u263a",
		MultiLine: "There are pros and cons to every alternate timeline. Fun facts about this " +
			"one: It has giant, telpathic spiders, eleven 9-11s, and the best ice cream in " +
			"the multiverse!",
		Content: "<article>\n" +
			"    You know the worst part about inventing teleportation?\n" +
			"    Suddenly, you're able to travel the whole galaxy, and the\n" +
			"    first thing you learn is, you're the last guy to invent\n" +
			"    teleportation.\n" +
			"</article>",
	}

	testNumeric = numStruct{
		Int8:    127,
		Int16:   32767,
		Int32:   2147483647,
		Int:     9223372036854775807,
		Int64:   9223372036854775807,
		Uint8:   255,
		Uint16:  65535,
		Uint32:  4294967295,
		Uint:    18446744073709551615,
		Uint64:  18446744073709551615,
		Float32: math.MaxFloat32,
		Float64: math.MaxFloat64,
	}

	testAbrev = numAbbrevStruct{
		Ki: 1000,
		Mi: 2048000000,
		Gi: 3000000000,
		Ti: 1000000000000,
		Pi: 4000000000000000,
		Ei: 9000000000000000000,
		Kf: 2200,
		Mf: 2300000,
		Gf: 2400000000,
		Tf: 3100000000000,
		Pf: 4200000000000000,
		Ef: 5300000000000000000,
	}

	testBool = boolStruct{
		Bool1: true,
		Bool2: false,
		Bool3: true,
		Bool4: false,
		Bool5: true,
		Bool6: false,
		Bool7: true,
		Bool8: false,
	}

	testTime = timeStruct{
		OffsetDateTime: tm(utc_date, "2017-12-25 08:10:00 -0800"),
		DateTime:       tm(date_time, "2017-12-25 08:10:00"),
		DateOnly:       tm(date_fmt, "2017-12-25"),
		TimeOnly:       tm(time_fmt, "08:10:00"),
		OffsetTime:     tm(utc_time, "08:10:00 -0800"),
	}

	testStringMap = stringMap{
		"Key1": "String1",
		"Key2": "String2",
	}

	testIntMap = intMap{
		"Key1": 2147483647,
		"Key2": -2147483648,
	}

	testStructMap = structMap{
		"Key1": simpleStruct{"String1", 41},
		"Key2": simpleStruct{"String2", 42},
	}

	testTimeMap = timeMap{
		"Key1": tm(utc_date, "2017-12-25 08:10:00 -0800"),
		"Key2": tm(date_time, "2017-12-25 08:10:00"),
		"Key3": tm(date_fmt, "2017-12-25"),
		"Key4": tm(time_fmt, "08:10:00"),
		"Key5": tm(utc_time, "08:10:00 -0800"),
	}

	testConfig = testConfigX{
		PlainString:  testString.PlainString,
		QuotedString: testString.QuotedString,
		SpecialChars: testString.SpecialChars,
		MultiLine:    testString.MultiLine,
		Content:      testString.Content,

		Numeric:      testNumeric,
		Abreviations: testAbrev,
		Bools:        testBool,
		Times:        testTime,
		StringMap:    testStringMap,
		IntegerMap:   testIntMap,
		StructMap:    testStructMap,
		TimeMap:      testTimeMap,
		Nested:       testNested,
	}

}

func tm(f, ts string) time.Time {
	d, _ := time.Parse(f, ts)
	return d
}

func getStructValue(x interface{}, key string) (reflect.Value, bool) {
	var val reflect.Value
	v1 := reflect.ValueOf(x)
	if isStructPtr(x) {
		v1 = v1.Elem()
	}
	for i, n := 0, v1.NumField(); i < n; i++ {
		this_key := v1.Type().Field(i).Name
		if this_key != key {
			continue
		}
		return v1.Field(i), true
	}
	return val, false
}

// create an empty tempfile, close it, return the name
func createTempFile(prefix string) string {
	fh, err := ioutil.TempFile(TEMP_DIR, prefix+"_")
	if err != nil {
		log.Fatal(err)
	}
	fh.Close()
	// replace backslashes in case of Windows
	return strings.Replace(fh.Name(), `\`, `/`, -1)
}

func writeFile(file string, data []byte) {
	err := ioutil.WriteFile(file, data, 0660)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return false
	}
	return true
}
