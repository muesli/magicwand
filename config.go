package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type DBus struct {
	Object string `json:"object,omitempty"`
	Path   string `json:"path,omitempty"`
	Method string `json:"method,omitempty"`
	Value  string `json:"value,omitempty"`
}

type Action struct {
	Keycode string `json:"keycode,omitempty"`
	Exec    string `json:"exec,omitempty"`
	DBus    DBus   `json:"dbus,omitempty"`
}

type Rule struct {
	Device      *Device `json:"device,omitempty"`
	Application string  `json:"application,omitempty"`
	Keycode     string  `json:"keycode,omitempty"`
	HWheel      int32   `json:"hwheel"`
	Dial        int32   `json:"dial"`
	Action      Action  `json:"action"`
}
type Rules []Rule

type Device struct {
	Name string `json:"name,omitempty"`
	Dev  string `json:"dev,omitempty"`
}
type Devices []Device

type Config struct {
	Devices Devices `json:"devices"`
	Rules   Rules   `json:"rules"`
}

// LoadConfig loads config from filename
func LoadConfig(filename string) (Config, error) {
	config := Config{}

	j, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(j, &config)
	return config, err
}

// Save writes config as json to filename
func (c Config) Save(filename string) error {
	j, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, j, 0644)
}

func (r Rules) FilterByDevice(dev Device) Rules {
	var res Rules
	for _, v := range r {
		if v.Device == nil ||
			(len(dev.Name) > 0 && v.Device.Name == dev.Name) ||
			(len(dev.Dev) > 0 && v.Device.Dev == dev.Dev) {
			res = append(res, v)
		}
	}

	return res
}

func (r Rules) FilterByApplication(name string) Rules {
	var res Rules
	for _, v := range r {
		if v.Application == "" || v.Application == name {
			res = append(res, v)
		}
		if len(v.Application) > 0 && v.Application[0] == '!' {
			if v.Application[1:] != name {
				res = append(res, v)
			}
		}
	}

	return res
}

func (r Rules) FilterByHWheel(wheel int32) Rules {
	var res Rules
	for _, v := range r {
		if v.HWheel == 0 || v.HWheel == wheel {
			res = append(res, v)
		}
	}

	return res
}

func (r Rules) FilterByDial(dial int32) Rules {
	var res Rules
	for _, v := range r {
		if v.Dial == 0 || v.Dial == dial {
			res = append(res, v)
		}
	}

	return res
}

func (r Rules) FilterByKeycodes(pressed map[uint16]struct{}) Rules {
	var res Rules
	for _, v := range r {
		kk := strings.Split(v.Keycode, "-")
		match := true
		for _, k := range kk {
			if k == "" {
				continue
			}

			kc, err := strconv.Atoi(k)
			if err != nil {
				log.Fatalf("%s is not a valid keycode: %s", k, err)
			}

			if _, ok := pressed[uint16(kc)]; !ok {
				match = false
				break
			}
		}

		if match {
			res = append(res, v)
		}
	}

	return res
}
