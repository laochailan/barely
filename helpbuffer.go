// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"github.com/nsf/termbox-go"
)

// HelpBuffer displays a cheat sheet of all usable keybindings in the current context.
type HelpBuffer struct {
	bufferName string
}

func drawHelpSection(y int, name string) int {
	if _, ok := config.Bindings[name]; !ok {
		return y
	}

	for _, b := range config.Bindings[name].Key {
		printLine(2, y, b.KeyName, -1, -1)
		printLine(15, y, b.Command, -1, -1)
		printLine(15+len(b.Command)+1, y, strings.Join(b.Args, " "), -1, -1)
		y++
	}
	return y
}

// Draw draws the content of the buffer.
func (b *HelpBuffer) Draw() {
	printLine(0, 0, "Command Help", int(termbox.AttrBold), 0)
	y := 2
	printLine(0, y, "global bindings:", int(termbox.AttrBold), -1)
	y += 2
	y = drawHelpSection(y, "")
	y++
	printLine(0, y, b.bufferName+" bindings:", int(termbox.AttrBold), -1)
	y += 2
	y = drawHelpSection(y, b.bufferName)
}

// Title returns the title string of the buffer.
func (b *HelpBuffer) Title() string {
	return "on buffer \"" + b.bufferName + "\""
}

// Name returns the name of the buffer.
func (b *HelpBuffer) Name() string {
	return "help"
}

// Close closes the buffer.
func (b *HelpBuffer) Close() {
}

// HandleCommand handles buffer local commands.
func (b *HelpBuffer) HandleCommand(cmd string, args []string, stack *BufferStack) bool {
	return false
}
