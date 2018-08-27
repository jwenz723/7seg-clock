package main

import (
	"log"
	"time"

	i2c "github.com/d2r2/go-i2c"
)

var pack *i2c.I2C
var digitMap = map[string]uint8{
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

func main() {
	// Connect to seven segment
	i2c, err := i2c.NewI2C(0x70, 1)
	if err != nil {
		log.Fatal(err)
	}
	pack = i2c
	// Free I2C connection on exit
	defer pack.Close()

	Clear()

	//_, _ = pack.WriteBytes([]byte{0x00 & 0xFF, 0x06 & 0xFF})
	//_, _ = pack.WriteBytes([]byte{0x02 & 0xFF, 0x7D & 0xFF})
	//_, _ = pack.WriteBytes([]byte{0x06 & 0xFF, 0x6D & 0xFF})
	//_, _ = pack.WriteBytes([]byte{0x04 & 0xFF, 0x00 & 0xFF})

	//Write(0, "1")
	//Write(1, "2")
	//Write(2, "3")
	//Write(3, "4")

	//WriteString("1456")

	// Turn on the colon
	_, _ = pack.WriteBytes([]byte{0x04 & 0xFF, 0x02 & 0xFF})

	for {
		WriteString(time.Now().Format("1504"))
		time.Sleep(15 * time.Second)
	}
}

func Clear() {
	for i := range [5]int{} {
		_, err := pack.WriteBytes([]byte{byte(i * 2), 0x00 & 0xFF})
		if err != nil {
			log.Fatal(err)
		}
	}
}

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
