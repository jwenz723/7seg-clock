package main

import (
	"log"
	"time"

	"fmt"
	"net/http"

	i2c "github.com/d2r2/go-i2c"
	"github.com/julienschmidt/httprouter"
	rpio "github.com/stianeikeland/go-rpio/v4"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	cf       = kingpin.Flag("config", "Path to yaml config file.").Default("config.yaml").Short('c').String()
	dryI2C   = kingpin.Flag("dryI2C", "Use this to run without a real I2C interface.").Short('d').Bool()
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
	alarmTime time.Time
)

func main() {
	kingpin.Parse()
	config, err := NewConfig(*cf)
	if err != nil {
		panic(fmt.Errorf("error parsing config file: %s", err))
	}

	alarmTime, err := time.ParseInLocation("15:04", config.AlarmTime, time.Now().Location())
	if err != nil {
		panic(fmt.Errorf("error parsing AlarmTime from config: %s", err))
	}

	if !*dryI2C {
		// Connect to seven segment
		i, err := i2c.NewI2C(config.I2CAddr, config.I2CBus)
		if err != nil {
			log.Fatal(err)
		}
		pack = i
		// Free I2C connection on exit
		defer pack.Close()
	} else {
		log.Println("dry run I2C")
	}

	// Initialize the display
	Begin()

	Clear()

	// Turn on the colon
	setColon(true)

	// A go routine to handle writing to the 7seg display
	writer := make(chan string)
	go func() {
		for s := range writer {
			WriteString(s)
		}
	}()

	alarmTime := time.Now()
	lastAlarmOccurrence := ""
	buttonPress := make(chan int)
	cancel := false

	go func() {
		l := ""
		for {
			select {
			case op := <-buttonPress:
				cancel = true
				switch op {
				case 1:
					alarmTime = alarmTime.Add(1 * time.Minute)
				case -1:
					alarmTime = alarmTime.Add(-1 * time.Minute)
				}

				l = alarmTime.Format("1504")
				writer <- l
			case <-time.After(1 * time.Second):
				n := time.Now()
				s := n.Format("1504")
				if l != s {
					if s == alarmTime.Format("1504") && n.Format("15042") != lastAlarmOccurrence {
						lastAlarmOccurrence = n.Format("15042")
						cancel = false
						go func() {
							b := byte(0)
							for !cancel {
								if b == 0 {
									b = byte(15)
								} else {
									b = 0
								}
								SetBrightness(b)
								time.Sleep(500 * time.Millisecond)
							}
							// Set back to default brightness
							SetBrightness(1)
						}()
					} else {
						l = s
						writer <- l
					}
				}

			}
		}
	}()

	err = rpio.Open()
	if err != nil {
		panic(err)
	}
	defer rpio.Close()

	pinInc := initPin(18)
	pinDec := initPin(23)
	pinAlarm := initPin(24)

	go func() {
		lastSend := time.Now()
		for {
			// if alarm pin is being pressed
			if pinAlarm.Read() == 0 {
				v := 0
				if pinInc.Read() == 0 {
					v = 1
				} else if pinDec.Read() == 0 {
					v = -1
				}

				if time.Since(lastSend) > (500 * time.Millisecond) {
					lastSend = time.Now()
					buttonPress <- v
				}
			}
			time.Sleep(10 * time.Millisecond)

			//			if t == alarmTime {
			//				for k, v := range config.AlarmTriggers {
			//					_, err := http.Get("https://maker.ifttt.com/trigger/" + k + "/with/key/" + v)
			//					if err != nil {
			//						fmt.Errorf("error with AlarmTrigger %s: %s\n", k, err)
			//						continue
			//					}
			//					fmt.Printf("AlarmTrigger %s success\n", k)
			//				}
			//
			//			}
		}
	}()

	router := httprouter.New()
	router.GET("/alarm/:time", alarm)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func initPin(id int) rpio.Pin {
	p := rpio.Pin(id)
	p.Input()
	p.PullUp()
	return p
}

// a handler to set the alarm time
func alarm(w http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	t, err := time.ParseInLocation("15:04", p.ByName("time"), time.Now().Location())
	if err != nil {
		fmt.Fprintf(w, "Failed to parse time. Must be specified in format 15:04.\nError: %s\n", err)
	}

	fmt.Fprintf(w, "Alarm: %s\n", t)
}

func setColon(on bool) {
	if pack != nil {
		_, _ = pack.WriteBytes([]byte{0x04 & 0xFF, 0x02 & 0xFF})
	} else {
		log.Println("dry run - setColon")
	}
}

// Begin will initialize driver with LEDs enabled and all turned off.
func Begin() {
	// TODO: is WriteRegU8 the same as python's _device.writeList ??
	// TODO: is 0x00 the same as python's [] ??

	// Turn on the oscillator.
	// self._device.writeList(HT16K33_SYSTEM_SETUP | HT16K33_OSCILLATOR, [])
	pack.WriteRegU8(0x20|0x01, 0x00)

	// Turn display on with no blinking.
	// self.set_blink(HT16K33_BLINK_OFF)
	pack.WriteRegU8(0x80|0x01|0x00, 0x00)

	// Set display to full brightness.
	// self.set_brightness(15)
	//   - > self._device.writeList(HT16K33_CMD_BRIGHTNESS | brightness, [])
	//pack.WriteRegU8(0xE0|1, 0x00)
	SetBrightness(1)
}

func SetBrightness(b byte) {
	if b < 0 || b > 15 {
		return
	}

	pack.WriteRegU8(0xE0|b, 0x00)
}

// Clear will clear the 7-Segment display
func Clear() {
	if pack != nil {
		for i := range [5]int{} {
			_, err := pack.WriteBytes([]byte{byte(i * 2), 0x00 & 0xFF})
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		log.Println("dry run - Clear()")
	}
}

// Write writes c to position pos on the 7-Segment display. pos must be between 0 and 3,
// where 0 is the far left segment and 3 is the far right segment.
func Write(pos int, c string) {
	if pack != nil {
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
	} else {
		log.Printf("dry run - Write %d = %s\n", pos, c)
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
