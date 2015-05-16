// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"github.com/nsf/termbox-go"
)

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

func (b *HelpBuffer) Title() string {
	return "on buffer \"" + b.bufferName + "\""
}

func (b *HelpBuffer) Name() string {
	return "help"
}

func (b *HelpBuffer) Close() {
}

func (b *HelpBuffer) HandleCommand(cmd string, args []string, stack *BufferStack) bool {
	return false
}
