package leds

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// https://scene-si.org/2016/07/19/building-your-own-build-status-indicator-with-golang-and-rpi3/

// Known Cloud Key LED names (regular + rack mount variants)
var KnownLEDs = []string{"blue", "white", "rack:blue", "rack:white", "ulogo_ctrl"}

// LED is an individual led
type LED struct {
	name string
}

// Name returns the LED name
func (r LED) Name() string {
	return r.name
}

// filename returns the /sys path of the led
func (r LED) filename() string {
	return "/sys/class/leds/" + r.name
}

// Exists checks if the LED exists on this system
func (r LED) Exists() bool {
	_, err := os.Stat(r.filename())
	return err == nil
}

func (r LED) read(where string) ([]byte, error) {
	filename := r.filename() + "/" + where
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (r LED) write(where, what string) LED {
	if !r.Exists() {
		return r
	}
	filename := r.filename() + "/" + where
	os.WriteFile(filename, []byte(what), 0666)
	return r
}

// On turns on the led to maximum brightness, and clears the current running trigger (if any)
func (r LED) On() LED {
	if !r.Exists() {
		return r
	}
	r.write("trigger", "none")
	maxBytes, err := r.read("max_brightness")
	if err != nil {
		return r.write("brightness", "255")
	}
	max := strings.TrimSuffix(string(maxBytes), "\n")
	return r.write("brightness", max)
}

// Off turns off the led, sets to zero brightness, and clears the current running trigger (if any)
func (r LED) Off() LED {
	if !r.Exists() {
		return r
	}
	r.write("trigger", "none")
	return r.write("brightness", "0")
}

// Brightness sets the brightness directly, and clears the current running trigger (if any)
func (r LED) Brightness(i int) LED {
	if !r.Exists() {
		return r
	}
	r.write("trigger", "none")
	return r.write("brightness", strconv.Itoa(i))
}

// Blink creates a blinking trigger action
func (r LED) Blink(i int, onTime int, offTime int) LED {
	if !r.Exists() {
		return r
	}
	r.write("trigger", "none")
	r.Brightness(i)
	r.write("trigger", "timer")
	r.write("delay_on", strconv.Itoa(onTime))
	r.write("delay_off", strconv.Itoa(offTime))
	return r
}

// LEDS is a controller for managing multiple LEDs
type LEDS struct{}

// LED returns an LED by name
func (r LEDS) LED(name string) LED {
	return LED{name: name}
}

// AllOff turns off all known Cloud Key LEDs (handles missing LEDs gracefully)
func (r LEDS) AllOff() {
	for _, name := range KnownLEDs {
		r.LED(name).Off()
	}
}

// DiscoverLEDs returns a list of LEDs that exist on this system
func DiscoverLEDs() []string {
	var found []string
	for _, name := range KnownLEDs {
		led := LED{name: name}
		if led.Exists() {
			found = append(found, name)
		}
	}
	return found
}

// PrintDiscoveredLEDs prints which LEDs were found on this system
func PrintDiscoveredLEDs() {
	found := DiscoverLEDs()
	if len(found) > 0 {
		fmt.Printf("Discovered LEDs: %s\n", strings.Join(found, ", "))
	} else {
		fmt.Println("No LEDs discovered")
	}
}
