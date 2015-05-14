// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"

	"github.com/notmuch/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

// A Buffer is a screen of the ui. Buffers are opened on a stack.
type Buffer interface {
	// Draw content on the screen
	Draw()
	// Title displayed in the bottom bar. It has to identify buffer uniquely.
	Title() string
	// Name of the buffer
	Name() string
	// Close the Buffer
	Close()

	// HandleCommand handles string commands. If it returns bool, the
	// command was accepted. If not, it was invalid.
	HandleCommand(cmd string, args []string, stack *BufferStack) bool
}

// BufferStack is the Stack structure managing drawing of the screen and
// buffers.
type BufferStack struct {
	buffers []Buffer

	prompt Prompt
}

func invalidCommand(cmd string) {
	StatusLine = "invalid command: " + cmd
}

func (b *BufferStack) Init(db *notmuch.Database) {
	fields := strings.Fields(config.General.Initial_Command)
	if len(fields) > 0 {
		accept := b.handleCommand(fields[0], fields[1:], db)
		if !accept {
			invalidCommand(fields[0])
		}
		if len(b.buffers) == 0 {
			b.Push(NewSearchBuffer("", nil))
		}
	}
}

func (b *BufferStack) Push(n Buffer) {
	for i, buf := range b.buffers {
		// If the buffer already exists, change to it instead.
		if buf.Name() == n.Name() && buf.Title() == n.Title() {
			n.Close()
			copy(b.buffers[i:], b.buffers[i+1:])
			b.buffers[len(b.buffers)-1] = buf
			b.refresh()
			return
		}
	}
	b.buffers = append(b.buffers, n)
	b.refresh()
}

// This string will be displayed at the bottom of the screen. useful for error messages.
var StatusLine string

func (b *BufferStack) Pop() {
	if len(b.buffers) == 0 {
		return
	}
	b.buffers[len(b.buffers)-1].Close()
	b.buffers = b.buffers[:len(b.buffers)-1]
	b.refresh()
}

func (b *BufferStack) refresh() {
	termbox.Clear(0, 0)
	if len(b.buffers) == 0 {
		return
	}
	b.buffers[len(b.buffers)-1].Draw()
	w, h := termbox.Size()
	cbuf := termbox.CellBuffer()
	for i := 0; i < w; i++ {
		cbuf[(h-2)*w+i].Bg = termbox.Attribute(config.Theme.BottomBar)
		cbuf[(h-2)*w+i].Fg = termbox.AttrBold
	}
	title := b.buffers[len(b.buffers)-1].Title()
	name := b.buffers[len(b.buffers)-1].Name()
	printLine(0, h-2, fmt.Sprintf("[%d: %s] %s", len(b.buffers)-1, name, title), -1, -1)
	if b.prompt.Active() {
		b.prompt.Draw()
	} else if StatusLine != "" {
		_, h := termbox.Size()
		printLine(0, h-1, StatusLine, -1, -1)
	}
}

func (b *BufferStack) handleCommand(cmd string, args []string, db *notmuch.Database) bool {
	switch cmd {
	case "close":
		b.Pop()
	case "quit":
		for len(b.buffers) > 0 {
			b.Pop()
		}
	case "search":
		b.Push(NewSearchBuffer(strings.Join(args, " "), db))
	case "prompt":
		StatusLine = ""
		b.prompt.Activate(strings.Join(args, " "))
	default:
		return false
	}
	return true
}

func (b *BufferStack) HandleEvent(event *termbox.Event, db *notmuch.Database) {
	if len(b.buffers) == 0 {
		return
	}
	if event.Type == termbox.EventResize {
		termbox.Flush()
		for _, buf := range b.buffers {
			buf.HandleCommand("resize", nil, b)
		}
		b.refresh()
		return
	}

	if event.Type == termbox.EventKey {
		if b.prompt.Active() {
			cmd, args := b.prompt.HandleEvent(event)
			if len(cmd) != 0 {
				accept := b.buffers[len(b.buffers)-1].HandleCommand(cmd, args, b)
				if !accept {
					accept = b.handleCommand(cmd, args, db)
				}
				if !accept {
					invalidCommand(cmd)
				}
				_, h := termbox.Size()
				printLine(0, h-1, StatusLine, -1, -1)
			}
			return
		}
		cmd := getBinding("", event.Ch, event.Key)
		accept := false
		if cmd == nil {
			cmd := getBinding(b.buffers[len(b.buffers)-1].Name(), event.Ch, event.Key)
			if cmd == nil {
				return
			}
			accept = b.buffers[len(b.buffers)-1].HandleCommand(cmd.Command, cmd.Args, b)
		} else {
			accept = b.handleCommand(cmd.Command, cmd.Args, db)
		}
		if !accept {
			invalidCommand(cmd.Command)
		}

		if StatusLine != "" {
			_, h := termbox.Size()
			printLine(0, h-1, StatusLine, -1, -1)
		}
	}

}
