[![MarkMueller](https://img.shields.io/badge/coverage-100%25-orange.svg)](http://markmueller.com/)
[![GoDoc](https://godoc.org/github.com/mkmueller/config?status.svg)](https://godoc.org/github.com/mkmueller/config)
[![GoDoc](https://img.shields.io/badge/example-configuration%20file-blue.svg)](http://htmlpreview.github.io/?https://github.com/mkmueller/config/blob/master/example-conf.html)


# config
`import "github.com/mkmueller/config"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>
Config provides encoding and decoding routines for configuration files. This
package supports most of the built-in datatypes, including string, int8-64,
uint8-64, float32-64, time.Time, struct, and string-keyed maps. Deeply nested
structs are supported as well as maps of structs. The data types not supported
are complex64/128 and slices.

This package also provides a Parse function which will allow any configuration
data to be parsed directly into a string map.

At this writing, struct tags are not supported. However, optional flags provide
a means to convert all fields to lower case or snake_case for encoding and
decoding.




## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [func Decode(x interface{}, src interface{}, options ...int) error](#Decode)
* [func DecodeFile(filename string, x interface{}, options ...int) error](#DecodeFile)
* [func Encode(x interface{}, options ...int) ([]byte, error)](#Encode)
* [func EncodeToFile(x interface{}, filename string, options ...int) error](#EncodeToFile)
* [type Decoder](#Decoder)
  * [func NewDecoder(x interface{}, options ...int) *Decoder](#NewDecoder)
  * [func (o *Decoder) DecodeBytes(bs []byte) error](#Decoder.DecodeBytes)
  * [func (o *Decoder) DecodeFile(filename string) error](#Decoder.DecodeFile)
  * [func (o *Decoder) DecodeStream(r io.Reader) error](#Decoder.DecodeStream)
  * [func (o *Decoder) DecodeString(s string) error](#Decoder.DecodeString)
* [type Encoder](#Encoder)
  * [func NewEncoder(x interface{}, options ...int) *Encoder](#NewEncoder)
  * [func (o *Encoder) ToBytes(bs *[]byte) error](#Encoder.ToBytes)
  * [func (o *Encoder) ToFile(filename string) error](#Encoder.ToFile)
  * [func (o *Encoder) ToStream(w io.Writer) error](#Encoder.ToStream)
* [type Parser](#Parser)
  * [func NewParser(options ...int) *Parser](#NewParser)
  * [func (o *Parser) Includes() []string](#Parser.Includes)
  * [func (o *Parser) Parse(bs []byte) (StringMap, error)](#Parser.Parse)
  * [func (o *Parser) ParseStream(r io.Reader) (StringMap, error)](#Parser.ParseStream)
* [type StringMap](#StringMap)
  * [func Parse(src interface{}, options ...int) (StringMap, error)](#Parse)
  * [func ParseFile(filename string, options ...int) (StringMap, error)](#ParseFile)

#### <a name="pkg-examples">Examples</a>
* [Decode](#example_Decode)
* [Decode (Map)](#example_Decode_map)
* [Encode](#example_Encode)
* [Encode (Map)](#example_Encode_map)
* [NewDecoder](#example_NewDecoder)
* [NewEncoder](#example_NewEncoder)

#### <a name="pkg-files">Package files</a>
[decode.go](/src/github.com/mkmueller/config/decode.go) [encode.go](/src/github.com/mkmueller/config/encode.go) [parser.go](/src/github.com/mkmueller/config/parser.go)


## <a name="pkg-constants">Constants</a>
``` go
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
```



## <a name="Decode">func</a> [Decode](/src/target/decode.go?s=3648:3713#L129)
``` go
func Decode(x interface{}, src interface{}, options ...int) error
```
Decode will accept a string, byte slice, or anything that implements an io.Reader



## <a name="DecodeFile">func</a> [DecodeFile](/src/target/decode.go?s=5480:5549#L208)
``` go
func DecodeFile(filename string, x interface{}, options ...int) error
```
DecodeFile will decode the supplied file into the supplied
struct. Decoder options are optional.



## <a name="Encode">func</a> [Encode](/src/target/encode.go?s=1994:2052#L93)
``` go
func Encode(x interface{}, options ...int) ([]byte, error)
```


## <a name="EncodeToFile">func</a> [EncodeToFile](/src/target/encode.go?s=2204:2275#L101)
``` go
func EncodeToFile(x interface{}, filename string, options ...int) error
```



## <a name="Decoder">type</a> [Decoder](/src/target/decode.go?s=2191:2350#L66)
``` go
type Decoder struct {
    // contains filtered or unexported fields
}
```
The Decoder converts the parsed data to the expected data type and assignes it to a struct.







### <a name="NewDecoder">func</a> [NewDecoder](/src/target/decode.go?s=2433:2488#L79)
``` go
func NewDecoder(x interface{}, options ...int) *Decoder
```
NewDecoder accepts a pointer to a struct or a map and returns a new Decoder.





### <a name="Decoder.DecodeBytes">func</a> (\*Decoder) [DecodeBytes](/src/target/decode.go?s=3271:3317#L115)
``` go
func (o *Decoder) DecodeBytes(bs []byte) error
```
DecodeBytes will accept a byteslice




### <a name="Decoder.DecodeFile">func</a> (\*Decoder) [DecodeFile](/src/target/decode.go?s=4116:4167#L144)
``` go
func (o *Decoder) DecodeFile(filename string) error
```
DecodeFile will decode the supplied filename




### <a name="Decoder.DecodeStream">func</a> (\*Decoder) [DecodeStream](/src/target/decode.go?s=3120:3169#L108)
``` go
func (o *Decoder) DecodeStream(r io.Reader) error
```
DecodeStream will accept an io.Reader




### <a name="Decoder.DecodeString">func</a> (\*Decoder) [DecodeString](/src/target/decode.go?s=3435:3481#L122)
``` go
func (o *Decoder) DecodeString(s string) error
```
DecodeString will accept a string




## <a name="Encoder">type</a> [Encoder](/src/target/encode.go?s=339:501#L21)
``` go
type Encoder struct {
    // contains filtered or unexported fields
}
```
The Encoder handles encoding a struct to an io.Writer.







### <a name="NewEncoder">func</a> [NewEncoder](/src/target/encode.go?s=568:623#L31)
``` go
func NewEncoder(x interface{}, options ...int) *Encoder
```
NewEncoder accepts a struct or map and returns a new Encoder.





### <a name="Encoder.ToBytes">func</a> (\*Encoder) [ToBytes](/src/target/encode.go?s=2343:2386#L106)
``` go
func (o *Encoder) ToBytes(bs *[]byte) error
```
ToBytes




### <a name="Encoder.ToFile">func</a> (\*Encoder) [ToFile](/src/target/encode.go?s=1365:1412#L63)
``` go
func (o *Encoder) ToFile(filename string) error
```
ToFile will encode a struct to the supplied filename. If the file exists,
it will not be overwritten unless the overwrite options is used.




### <a name="Encoder.ToStream">func</a> (\*Encoder) [ToStream](/src/target/encode.go?s=2482:2527#L114)
``` go
func (o *Encoder) ToStream(w io.Writer) error
```
ToStream




## <a name="Parser">type</a> [Parser](/src/target/parser.go?s=1338:1486#L56)
``` go
type Parser struct {
    // contains filtered or unexported fields
}
```
The Parser handles parsing input data from a reader.







### <a name="NewParser">func</a> [NewParser](/src/target/parser.go?s=2402:2440#L95)
``` go
func NewParser(options ...int) *Parser
```
NewParser returns a new Parser.





### <a name="Parser.Includes">func</a> (\*Parser) [Includes](/src/target/parser.go?s=8358:8394#L377)
``` go
func (o *Parser) Includes() []string
```
Includes will return a list of file names that have been included in the
source configuration file.




### <a name="Parser.Parse">func</a> (\*Parser) [Parse](/src/target/parser.go?s=3680:3732#L146)
``` go
func (o *Parser) Parse(bs []byte) (StringMap, error)
```
Parse a byte slice to a string map.




### <a name="Parser.ParseStream">func</a> (\*Parser) [ParseStream](/src/target/parser.go?s=3816:3876#L151)
``` go
func (o *Parser) ParseStream(r io.Reader) (StringMap, error)
```
Parse a stream to a string map.




## <a name="StringMap">type</a> [StringMap](/src/target/parser.go?s=1553:1585#L67)
``` go
type StringMap map[string]string
```
Type StringMap is the data type output by the Parse function.







### <a name="Parse">func</a> [Parse](/src/target/parser.go?s=2754:2816#L111)
``` go
func Parse(src interface{}, options ...int) (StringMap, error)
```
Parse a string, a byte slice or an io.Reader to a string map.


### <a name="ParseFile">func</a> [ParseFile](/src/target/parser.go?s=3141:3207#L123)
``` go
func ParseFile(filename string, options ...int) (StringMap, error)
```
Parse a file

- - -
