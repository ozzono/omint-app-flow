package main

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ozzono/normalize"

	adb "github.com/ozzono/adbtools"
)

var (
	device adb.Device
)

const (
	omint         = "br.com.omint.apps.minhaomint"
	omintActivity = "br.com.omint.apps.minhaomint.MainActivity"
)

func main() {
	devices, err := adb.Devices()
	if err != nil && !strings.Contains(err.Error(), "no devices found") {
		log.Println(err)
		return
	}
	if len(devices) == 0 {
		log.Println("0 devices found")
		err = adb.StartAnbox()
		if err != nil {
			log.Printf("StartAnbox err: %v", err)
			return
		}
		sleep(20, "starting anbox")
		devices, err = adb.Devices()
		if err != nil {
			log.Println(err)
			return
		}
	}
	device = devices[0]
	device.CloseApp(omint)
	device.StartApp(omint, omintActivity, "")
	if !device.WaitApp(omint, 1000, 10) {
		return
	}
	c := 0
	for !hasInScreen("login") {
		sleep(10, "waiting for login in xml screen")
		c++
		if c >= 10 {
			return
		}
	}
	fmt.Printf("%#v\n", applyRegexp("loginr.*?(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])", device.XMLScreen(false)))
}

func sleep(t int, note string) {
	if len(note) > 0 {
		log.Println("note:", note)
	}
	time.Sleep(time.Duration(int64(t*100)) * time.Second)
}

func hasInScreen(content ...string) bool {
	for _, item := range content {
		if strings.Contains(normalize.Norm(strings.ToLower(device.XMLScreen(true))), normalize.Norm(strings.ToLower(item))) {
			return true
		}
	}
	return false
}

func applyRegexp(exp, text string) []string {
	if device.Log {
		log.Printf("Applying expression: %s", exp)
	}
	re := regexp.MustCompile(exp)
	match := re.FindStringSubmatch(text)
	if len(match) < 1 {
		fmt.Printf("Unable to find match for exp %s\n", exp)
		return []string{}
	}
	return match
}

func newRandNumber(i int) int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Intn(i)
}

func xml2tap(xmlcoords string) (int, int) {
	if device.Log {
		log.Printf("Parsing coords: %s", xmlcoords)
	}
	openbracket := "["
	closebracket := "]"
	joinedbracket := "]["
	if string(xmlcoords[0]) == openbracket && string(xmlcoords[len(xmlcoords)-1]) == closebracket && strings.Contains(xmlcoords, joinedbracket) {
		stringcoords := strings.Split(xmlcoords, "][")
		leftcoords := strings.Split(string(stringcoords[0][1:]), ",")
		rightcoords := strings.Split(string(stringcoords[1][:len(stringcoords[1])-1]), ",")
		x1, err := strconv.Atoi(leftcoords[0])
		if err != nil {
			fmt.Printf("atoi err: %v", err)
			return 0, 0
		}
		y1, err := strconv.Atoi(leftcoords[1])
		if err != nil {
			fmt.Printf("atoi err: %v", err)
			return 0, 0
		}
		x2, err := strconv.Atoi(rightcoords[0])
		if err != nil {
			fmt.Printf("atoi err: %v", err)
			return 0, 0
		}
		y2, err := strconv.Atoi(rightcoords[1])
		if err != nil {
			fmt.Printf("atoi err: %v", err)
			return 0, 0
		}
		x := (x1 + x2) / 2
		y := (y1 + y2) / 2
		fmt.Printf("%s --- x: %d y: %d\n", xmlcoords, x, y)
		return x, y
	}
	return 0, 0
}
