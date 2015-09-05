// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"errors"
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

// KeyBinding represents a single keybinding.
type KeyBinding struct {
	Ch      rune
	Key     termbox.Key // for nonprintables
	KeyName string

	Command string
	Args    []string
}

// UnmarshalText implements the encoding.TextUnmarshaller interface.
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

// KeyBindings represent a set of keybindings.
type KeyBindings struct {
	Key []*KeyBinding
}

// Account represents a mail account set up to send mail.
type Account struct {
	Addr             string
	Sendmail_Command string
	Sent_Tag         []string
	Sent_Dir         string
	Draft_Dir        string
}

// TagAlias represents an alias for tags.
type TagAlias struct {
	tag   string
	alias string
}

// UnmarshalText implements the encoding.TextUnmarshaller interface.
func (t *TagAlias) UnmarshalText(text []byte) error {
	str := string(text)
	fields := strings.Fields(str)
	if len(fields) > 2 || len(fields) == 0 {
		return errors.New("Tag aliases must be of form 'tag alias'.")
	}

	t.tag = fields[0]
	if len(fields) == 2 {
		t.alias = fields[1]
	}
	return nil
}

// Config holds all configuration values.
// Refer to gcfg documentation for the resulting config file syntax.
type Config struct {
	General struct {
		Database          string
		Initial_Command   string
		Synchronize_Flags bool
	}

	Bindings map[string]*KeyBindings

	Theme struct {
		BottomBar int
		Date      int
		Subject   int
		From      int
		Tags      int

		Error int

		HlBg int
		HlFg int

		Quote int
	}

	Commands struct {
		Attachments string
		Editor      string
	}

	Account map[string]*Account

	Tags struct {
		Alias []*TagAlias
	}
}

// PostConfig contains post processed config fields, e.g. values
// stored in maps for faster access
type PostConfig struct {
	TagAliases map[string]string
}

const (
	configPath = "$HOME/.config/barely/config"
)

var config Config
var pconfig PostConfig

// default configuration
const DefaultCfg = `# This is the default configuration file for barely.
# barely looks for it in '~/.config/barely/config'
#
# Omitted options will default to the settings they have here.
# For syntax, see http://git-scm.com/docs/git-config#_syntax

[general]
# Location of the notmuch database
database=~/mail
# First command to be executed on start. This should open a
# new buffer. If it doesn't, a search buffer for "" is opened.
initial-command=msearch tag:unread
# Whether barely should add matching maildir tags after changing
# message tags.
synchronize-flags=true

# For every address you want to send mail with, there has to be an
# account section like this one. the addr, sendmail-command and
# sent-dir are mandatory for sending.
# draft-dir is mandatory for saving drafts of course.
#
# [account "example"]
# addr = example@example.com
# sendmail-command = msmtp --account=example -t
# sent-dir = $HOME/mail/example/sent
# draft-dir = $HOME/mail/example/draft
# sent-tag = sent
# sent-tag = example

[commands]
# program used to open all tpyes of attachments
attachments=xdg-open
# editor program
editor=vim

# This section describes the color theme. Colors are numbers
# in the terminal 256 color cube.
[theme]
bottombar = 241

date = 103
subject = 110
from = 115
tags = 244

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
key = @ refresh

[bindings "search"]
key = up move up
key = down move down
key = pageup move pageup
key = pagedown move pagedown
key = enter show
key = s untag unread
key = & tag deleted

[bindings "mail"]
key = up move up
key = down move down
key = pageup move pageup
key = pagedown move pagedown
key = enter show
key = r reply
key = / prompt search
key = | prompt search
key = n search
key = N rsearch

[bindings "compose"]
key = up move up
key = down move down
key = pageup move pageup
key = pagedown move pagedown
key = enter edit
key = y send
key = a prompt attach
key = A deattach

# The tags section can be used to set display aliases for tags.
# This can be used to hide or abbreviate common tags.
#
# [tags]
# alias = replied >
# alias = attachment @
# alias = sent  # empty alias means hiding tag

`

func preparePostConfig(pcfg *PostConfig, cfg *Config) {
	pcfg.TagAliases = make(map[string]string)
	for _, a := range cfg.Tags.Alias {
		pcfg.TagAliases[a.tag] = a.alias
	}
}

// LoadConfig loads the configuration from the standard configuration file path and sets the
// global config struct.
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

	preparePostConfig(&pconfig, &config)
}

// getBinding returns a key binding fitting a pressed key (Ch, Key) for a specific section.
// If no such binding exists, it returns nil.
//
// Global bindings are associated to the section "".
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

// getAccount fetches an account for a given mail address.
func getAccount(addr string) *Account {
	for _, val := range config.Account {
		if val.Addr == addr {
			return val
		}
	}
	return nil
}
