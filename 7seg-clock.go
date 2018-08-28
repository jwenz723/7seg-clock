package main

import (
	"log"
	"time"

	i2c "github.com/d2r2/go-i2c"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"fmt"
)

var (
	cf       = kingpin.Flag("config", "Path to yaml config file.").Default("config.yaml").Short('c').String()
	pack     *i2c.I2C
	digitMap = map[string]uint8{
		" ": 0x00,
		"-": 0x40,
		"0": 0x3F,
		"1": 0x06,
		"2": 0x5B,
		"3": 0x4F,
		"4": 0x66,
		"5": 0x6D,
		"6": 0x7D,
		"7": 0x07,
		"8": 0x7F,
		"9": 0x6F,
		"A": 0x77,
		"B": 0x7C,
		"C": 0x39,
		"D": 0x5E,
		"E": 0x79,
		"F": 0x71,
	}
)

func main() {
	kingpin.Parse()
	config, err := NewConfig(*cf)
	if err != nil {
		panic(fmt.Errorf("error parsing config file: %s", err))
	}

	// Connect to seven segment
	i2c, err := i2c.NewI2C(config.I2CAddr, config.I2CBus)
	if err != nil {
		log.Fatal(err)
	}
	pack = i2c
	// Free I2C connection on exit
	defer pack.Close()

	Clear()

	// Turn on the colon
	_, _ = pack.WriteBytes([]byte{0x04 & 0xFF, 0x02 & 0xFF})

	writer := make(chan string)
	go func() {
		for s := range writer {
			WriteString(s)
		}
	}()

	l := ""
	for {
		s := time.Now().Format("1504")
		if l != s {
			l = s
			writer <- l
		}

		time.Sleep(15 * time.Second)
	}
}

// Clear will clear the 7-Segment display
func Clear() {
	for i := range [5]int{} {
		_, err := pack.WriteBytes([]byte{byte(i * 2), 0x00 & 0xFF})
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Write writes c to position pos on the 7-Segment display. pos must be between 0 and 3,
// where 0 is the far left segment and 3 is the far right segment.
func Write(pos int, c string) {
	if pos < 0 || pos > 3 {
		return
	}

	offset := 0
	if pos >= 2 {
		offset = 1
	}

	_, err := pack.WriteBytes([]byte{byte((pos + offset) * 2), digitMap[c] & 0xFF})
	log.Printf("Writing %#x = %#x\n", byte((pos+offset)*2), digitMap[c]&0xFF)
	if err != nil {
		log.Fatal(err)
	}
}

// WriteString writes a string to the 7-Segment display. Nothing will happen if len(s) > 4.
func WriteString(s string) {
	if len(s) > 4 {
		return
	}

	r := []rune(s)

	pos := 3
	for i := len(r) - 1; i >= 0; i-- {
		Write(pos, string(r[i]))
		pos--
	}
}
