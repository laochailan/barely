// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"code.google.com/p/gcfg"
	termbox "github.com/nsf/termbox-go"
)

var specialKeys = map[string]termbox.Key{
	"up":       termbox.KeyArrowUp,
	"down":     termbox.KeyArrowDown,
	"left":     termbox.KeyArrowLeft,
	"right":    termbox.KeyArrowRight,
	"pageup":   termbox.KeyPgup,
	"pagedown": termbox.KeyPgdn,
	"enter":    termbox.KeyEnter,
}

type KeyBinding struct {
	Ch      rune
	Key     termbox.Key // for nonprintables
	KeyName string

	Command string
	Args    []string
}

func (k *KeyBinding) UnmarshalText(text []byte) error {
	str := string(text)
	fields := strings.Fields(str)

	if len(fields) < 2 {
		return fmt.Errorf("Expected syntax 'key command'")
	}
	if utf8.RuneCountInString(fields[0]) == 1 {
		k.Ch, _ = utf8.DecodeRuneInString(fields[0])
	} else {
		ok := false
		k.Key, ok = specialKeys[fields[0]]
		if !ok {
			return fmt.Errorf("Unsupported key '%s'", fields[0])
		}
	}
	k.KeyName = fields[0]

	k.Command = fields[1]
	k.Args = fields[2:]
	return nil
}

type KeyBindings struct {
	Key []*KeyBinding
}

// Config holds all configuration values.
// Refer to gcfg documentation for the resulting config file syntax.
type Config struct {
	General struct {
		Database        string
		Initial_Command string
	}

	Bindings map[string]*KeyBindings

	Theme struct {
		BottomBar int
		Date      int
		Subject   int
		From      int

		Error int

		HlBg int
		HlFg int

		Quote int
	}

	Commands struct {
		Attachments string
		Editor      string
	}
}

const (
	configPath = "$HOME/.config/barely/config"
)

var config Config

// default configuration
const DefaultCfg = `# This is the default configuration file for barely.
# barely looks for it in '~/.config/barely/config'
#
# Omitted options will default to the settings they have here.
# For syntax, see http://git-scm.com/docs/git-config#_syntax

[general]
# Location of the notmuch database
database="$HOME/mail"
# First command to be executed on start. This should open a
# new buffer. If it doesn't, a search buffer for "" is opened.
initial-command="search tag:unread"

[commands]
# program used to open all tpyes of attachments
attachments="xdg-open"
# editor program
editor="vim"

# This section describes the color theme. Colors are numbers
# in the terminal 256 color cube.
[theme]
bottombar = 241

date = 103
subject = 110
from = 115

error = 88

hlbg = 240
hlfg = 147

quote = 80

# The bindings sections contain keybinding definitions of the
# form
#	key = KEY COMMAND ARGS...
#
# Valid commands differ from buffer to buffer.

[bindings]
key = q quit
key = d close
key = / prompt search
key = : prompt
key = ? help

[bindings "search"]
key = up move up
key = down move down
key = pageup move pageup
key = pagedown move pagedown
key = enter show

[bindings "mail"]
key = up move up
key = down move down
key = pageup move pageup
key = pagedown move pagedown
key = enter show
key = r reply

[bindings "compose"]
key = up move up
key = down move down
key = pageup move pageup
key = pagedown move pagedown
key = enter edit
`

func LoadConfig() {
	err := gcfg.ReadStringInto(&config, DefaultCfg)
	if err != nil {
		panic(err)
	}
	path := os.ExpandEnv(configPath)
	err = gcfg.ReadFileInto(&config, path)
	if err != nil {
		fmt.Println(err)
	}
}

func getBinding(section string, Ch rune, Key termbox.Key) *KeyBinding {
	sec := config.Bindings[section]
	if sec == nil {
		return nil
	}
	keys := sec.Key

	for i := range keys {
		if (Ch != 0 && Ch == keys[i].Ch) || (Ch == 0 && Key == keys[i].Key) {
			return keys[i]
		}
	}
	return nil
}
