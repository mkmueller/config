package config_test

import (
	"bytes"
	"fmt"
	"github.com/mkmueller/config"
	"log"
)

// Encode a struct using lower case option.
func ExampleEncode() {
	x := struct {
		E  float64
		Pi float64
	}{2.718281828459, 3.14159265359}
	ba, err := config.Encode(x, config.ENCODE_LOWER_CASE)
	if err != nil {
		log.Print(err)
		return
	}
	fmt.Printf("%s", ba)

	// Output:
	// e = 2.718281828459
	// pi = 3.14159265359
}

// Encode a map of float64 values.
func ExampleEncode_map() {
	x := make(map[string]float64)
	x["e"] = 2.718281828459
	x["pi"] = 3.14159265359
	ba, err := config.Encode(x)
	if err != nil {
		log.Print(err)
		return
	}
	fmt.Printf("%s", ba)

	// Output:
	// e = 2.718281828459
	// pi = 3.14159265359
}

func ExampleNewEncoder() {
	x := struct{ Pi float64 }{3.14159265359}
	var ba []byte
	o := config.NewEncoder(x)
	o.ToBytes(&ba)
	fmt.Printf("%s", ba)

	// Output:
	// Pi = 3.14159265359

}

// The Encode method will encode directly to a byte slice or anything that
// implements an io.Writer
func ExampleEncoder_Encode() {
	var x struct{ Pi float64 }

	// Encode to a byte slice
	x.Pi = 3.14159265359
	var bs []byte
	err := config.NewEncoder(x).ToBytes(&bs)
	if logError(err) {
		return
	}
	fmt.Printf("%s", bs)

	// Encode to a byte buffer
	x.Pi = 3.141592653589
	var buf bytes.Buffer
	err = config.NewEncoder(x, config.ENCODE_LOWER_CASE).ToStream(&buf)
	if logError(err) {
		return
	}
	fmt.Printf("%s", buf.Bytes())

	// Output:
	// Pi = 3.14159265359
	// pi = 3.141592653589
}

// Decode float64 values to a map.
func ExampleDecode_map() {
	m := make(map[string]float64)
	cfg := `
	    e = 2.718281828459
	    pi = 3.14159265359
	`
	err := config.Decode(m, cfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(m["e"])
	fmt.Println(m["pi"])

	// Output:
	// 2.718281828459
	// 3.14159265359

}

func ExampleNewDecoder() {
	var x struct{ Pi float32 }
	cfg := "pi = 3.1415927"
	o := config.NewDecoder(&x, config.IGNORE_CASE)
	err := o.DecodeString(cfg)
	if logError(err) {
		return
	}
	fmt.Println(x.Pi)
	// Output: 3.1415927
}

// Decode a string
func ExampleDecoder_Decode() {
	var x struct {
		Pi               float32
		Question, Answer string
	}
	cfg := `
        pi = 3.1415927
        question = What is my purpose?
        answer = You pass butter.
        `
	o := config.NewDecoder(&x, config.IGNORE_CASE)
	if err := o.DecodeString(cfg); err != nil {
		log.Println(err)
	}
	fmt.Println(x.Pi)
	fmt.Println(x.Question)
	fmt.Println(x.Answer)

	// Output:
	// 3.1415927
	// What is my purpose?
	// You pass butter.
}

// Decode a string
func ExampleDecode() {
	var x struct {
		Pi               float32
		Question, Answer string
	}
	cfg := `
        pi = 3.1415927
        question = What is my purpose?
        answer = You pass butter.
        `
	if err := config.Decode(&x, cfg, config.IGNORE_CASE); err != nil {
		log.Println(err)
	}
	fmt.Println(x.Pi)
	fmt.Println(x.Question)
	fmt.Println(x.Answer)

	// Output:
	// 3.1415927
	// What is my purpose?
	// You pass butter.
}

func logError(err error) bool {
	if err != nil {
		log.Print(err)
		return true
	}
	return false
}
