# MagicWand

MagicWand makes your input devices context sensitive!

It lets you define application-specific keyboard shortcuts and mouse events to
trigger actions like keystroke emulation, command execution or even DBus method
calls.

This means you can make your horizontal wheel behave differently for different
applications: make it switch tabs when your browser is focused, but control the
volume when your music player is active.

Note: MagicWand currently only works on Linux with evdev and Xorg!

## Installation

Make sure you have a working Go environment (Go 1.7 or higher is required).
See the [install instructions](http://golang.org/doc/install.html).

To install MagicWand, simply run:

    go get github.com/muesli/magicwand

## Usage

Edit the config file and adjust it to your needs. First things first, tell
MagicWand which input devices to operate on. You can either match them by
their name or directly specify an input device file. For example:

```
  "devices": [
    {
      "name": "MX Master"
    },
    {
      "dev": "/dev/input/event6"
    }
  ]
```

This will match two devices and makes MagicWand react to input events from:

1.  Any device with the string "MX Master" in its name
2.  The device `/dev/input/event6`

Next tell MagicWand which events to react upon:

```
  "rules": [
    {
      "application": "Firefox",
      "hwheel": 1,
      "action": {
        "keycode": "42-29-15"
      }
    }
  ]
```

In plain english, this means: if the currently focused application is `Firefox`
and you `scroll left` with the horizontal mouse wheel, emulate a keyboard stroke
`Ctrl-Shift-Left` (keycodes 42, 29 and 15): this makes Firefox jump to the
previous tab.

If you have multiple devices emitting certain events, you can configure which
device a rule applies to:

```
  "rules": [
    {
      "device": {
        "name": "MX Master"
      },
      "hwheel": 1,
      "action": {
        ...
      }
    }
  ]
```

Another example would be defining global shortcuts for certain mouse buttons:

```
    {
      "application": "!Firefox",
      "keycode": "276",
      "action": {
          ...
      }
    }
```

This translates to: if the currently focused application is anything but
`Firefox` and the `forward` mouse button was pressed (keycode 276), trigger
an action.

Actions can not only be keystrokes (as in the example above), but you can also
use them to execute a command:

```
      "action": {
        "exec": "pactl set-sink-volume 0 +5%"
      }
```

Last but not least, actions let you call DBus methods:

```
      "action": {
        "dbus": {
          "object": "org.kde.KWin",
          "path": "/KWin",
          "method": "org.kde.KWin.previousDesktop"
        }
      }
```

Once you're done, start MagicWand:

```
$ magicwand
```

There are config examples for a few devices in the `configs` directory. You can
try them out by starting `magicwand` with the `-config` argument:

```
$ magicwand -config ./configs/logitech_mxmaster.json
```

## Development

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/muesli/magicwand)
[![Build Status](https://travis-ci.org/muesli/magicwand.svg?branch=master)](https://travis-ci.org/muesli/magicwand)
[![Go ReportCard](http://goreportcard.com/badge/muesli/magicwand)](http://goreportcard.com/report/muesli/magicwand)
