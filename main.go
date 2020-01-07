package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bendahl/uinput"
	"github.com/godbus/dbus"
	evdev "github.com/gvalkov/golang-evdev"
)

var (
	config Config

	dbusConn  *dbus.Conn
	keyboard  uinput.Keyboard
	threshold time.Time

	pressed = make(map[uint16]struct{})

	configFile = flag.String("config", "config.json", "path to config file")
	debug      = flag.Bool("debug", false, "enables debug output")
	timeout    = flag.Uint("threshold", 200, "threshold in ms between wheel events")
)

type Event struct {
	*evdev.InputEvent
	Device Device
}

// logs if debug is enabled
func dLog(a ...interface{}) {
	if !*debug {
		return
	}

	log.Println(a...)
}

// prints all detected input devices
func listDevices() error {
	devs, err := evdev.ListInputDevices()
	if err != nil {
		return fmt.Errorf("Can't find input devices")
	}

	for _, val := range devs {
		dLog("ID->", val.File.Name(), "Device->", val.Name)
	}

	return nil
}

// findDevice returns the device file matching an input device's name
func findDevice(s string) (string, error) {
	devs, err := evdev.ListInputDevices()
	if err != nil {
		return "", err
	}

	for _, val := range devs {
		if strings.Contains(val.Name, s) {
			return val.File.Name(), nil
		}
	}

	return "", fmt.Errorf("No such device")
}

// emulates a (multi-)key press
func emulateKeyPress(keys string) {
	kk := strings.Split(keys, "-")
	for i, k := range kk {
		kc, err := strconv.Atoi(k)
		if err != nil {
			log.Fatalf("%s is not a valid keycode: %s", k, err)
		}

		if i+1 < len(kk) {
			keyboard.KeyDown(kc)
			defer keyboard.KeyUp(kc)
		} else {
			keyboard.KeyPress(kc)
		}
	}
}

// executes a dbus method
func executeDBusMethod(object, path, method, args string) {
	call := dbusConn.Object(object, dbus.ObjectPath(path)).Call(method, 0)
	if call.Err != nil {
		log.Printf("dbus call failed: %s", call.Err)
	}
}

// executes a command
func executeCommand(cmd string) {
	args := strings.Split(cmd, " ")
	c := exec.Command(args[0], args[1:]...)
	if err := c.Start(); err != nil {
		panic(err)
	}
}

// executes an action
func executeAction(a Action) {
	dLog(fmt.Sprintf("Executing action: %+v", a))

	if a.Keycode != "" {
		emulateKeyPress(a.Keycode)
	}
	if a.DBus.Method != "" {
		executeDBusMethod(a.DBus.Object, a.DBus.Path, a.DBus.Method, a.DBus.Value)
	}
	if a.Exec != "" {
		executeCommand(a.Exec)
	}
}

// handles mouse wheel events
func mouseWheelEvent(ev Event, activeWindow Window) {
	rel := evdev.RelEvent{}
	rel.New(ev.InputEvent)

	if ev.Code != evdev.REL_HWHEEL && ev.Code != evdev.REL_DIAL {
		return
	}

	if time.Since(threshold) < time.Millisecond*time.Duration(*timeout) {
		dLog("Discarding wheel event below threshold")
		return
	}
	threshold = time.Now()

	switch ev.Code {
	case evdev.REL_HWHEEL:
		rr := config.Rules.FilterByDevice(ev.Device).FilterByDial(0).FilterByHWheel(ev.Value).FilterByKeycodes(pressed).FilterByApplication(activeWindow.Class)
		if len(rr) == 0 {
			return
		}

		executeAction(rr[0].Action)

	case evdev.REL_DIAL:
		rr := config.Rules.FilterByDevice(ev.Device).FilterByDial(ev.Value).FilterByHWheel(0).FilterByKeycodes(pressed).FilterByApplication(activeWindow.Class)
		if len(rr) == 0 {
			return
		}

		executeAction(rr[0].Action)

	default:
		// dLog(rel.String())
	}
}

// handles key events
func keyEvent(ev Event, activeWindow Window) {
	kev := evdev.KeyEvent{}
	kev.New(ev.InputEvent)

	pressed[ev.Code] = struct{}{}
	if kev.State != evdev.KeyUp {
		return
	}

	rr := config.Rules.FilterByDevice(ev.Device).FilterByHWheel(0).FilterByDial(0).FilterByKeycodes(pressed).FilterByApplication(activeWindow.Class)
	delete(pressed, ev.Code)

	if len(rr) == 0 {
		return
	}

	executeAction(rr[0].Action)
}

func handleEvent(ev Event, win Window) {
	switch ev.Type {
	case evdev.EV_KEY:
		// dLog("Key event:", ev.String())
		keyEvent(ev, win)
	case evdev.EV_REL:
		// dLog("Rel event:", ev.String())
		mouseWheelEvent(ev, win)
	case evdev.EV_ABS:
	case evdev.EV_SYN:
	case evdev.EV_MSC:
	case evdev.EV_LED:
	case evdev.EV_SND:
	case evdev.EV_SW:
	case evdev.EV_PWR:
	case evdev.EV_FF:
	case evdev.EV_FF_STATUS:
	default:
		log.Println("Unexpected event type:", ev.String())
	}
}

func subscribeToDevice(dev Device, keychan chan Event) {
	go func(dev Device) {
		for {
			var err error
			df := dev.Dev
			if df == "" {
				df, err = findDevice(dev.Name)
				if err != nil {
					log.Fatalf("Could not find device for %s", dev.Name)
				}
			}

			ed, err := evdev.Open(df)
			if err != nil {
				panic(err)
			}
			dLog(ed.String())

			for {
				ev, eerr := ed.ReadOne()
				if eerr != nil {
					log.Printf("Error reading from device: %v", eerr)
					time.Sleep(1 * time.Second)
					break
				}

				keychan <- Event{
					InputEvent: ev,
					Device:     dev,
				}
			}
		}
	}(dev)
}

func main() {
	var err error
	flag.Parse()
	config, err = LoadConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	dbusConn, err = dbus.SessionBus()
	if err != nil {
		panic(err)
	}

	x := Connect(os.Getenv("DISPLAY"))
	defer x.Close()

	tracker := make(chan Window)
	x.TrackActiveWindow(tracker, time.Second)
	go func() {
		for w := range tracker {
			dLog(fmt.Sprintf("Active window changed to %s (%s)", w.Class, w.Name))
		}
	}()

	keyboard, err = uinput.CreateKeyboard("/dev/uinput", []byte("Virtual Wand"))
	if err != nil {
		log.Fatalf("Could not create virtual input device (/dev/uinput): %s", err)
	}
	defer keyboard.Close()

	err = listDevices()
	if err != nil {
		panic(err)
	}

	keychan := make(chan Event)
	for _, dev := range config.Devices {
		subscribeToDevice(dev, keychan)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	var quit bool

	for {
		select {
		case <-sigchan:
			quit = true
		case ev := <-keychan:
			handleEvent(ev, x.ActiveWindow())
		}

		if quit {
			break
		}
	}

	fmt.Println("Shutting down...")
	if *debug {
		err = config.Save(*configFile)
		if err != nil {
			log.Fatal(err)
		}
	}
}
