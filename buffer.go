package main

import (
	"fmt"
	"strings"

	"github.com/notmuch/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

// A Buffer is a screen of the ui. Buffers are opened on a stack.
type Buffer interface {
	Draw()                                   // Draw content on the screen
	Title() string                           // title displayed in the bottom bar
	Name() string                            // name of the buffer
	Close()                                  // Close the Buffer
	HandleCommand(cmd string, args []string) // Handle a command
}

// BufferStack is the Stack structure managing drawing of the screen and
// buffers.
type BufferStack struct {
	buffers []Buffer

	prompt []rune
}

func (b *BufferStack) Push(n Buffer) {
	b.buffers = append(b.buffers, n)
	b.refresh()
}

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
	if b.prompt != nil {
		printLine(1, h-1, string(b.prompt), -1, -1)
	}
}

func (b *BufferStack) handlePrompt(e *termbox.Event, db *notmuch.Database) {
	if e.Ch == 0 {
		switch e.Key {
		case termbox.KeyEsc:
			b.prompt = nil
			b.refresh()
		case termbox.KeyEnter:
			fields := strings.Fields(string(b.prompt))
			if len(fields) > 0 {
				b.buffers[len(b.buffers)-1].HandleCommand(fields[0], fields[1:])
				b.handleCommand(fields[0], fields[1:], db)
			}
			b.prompt = nil
			b.refresh()
		case termbox.KeyBackspace2:
			if len(b.prompt) > 0 {
				b.prompt = b.prompt[:len(b.prompt)-1]
			}
		case termbox.KeySpace:
			b.prompt = append(b.prompt, ' ')
		}
	} else {
		b.prompt = append(b.prompt, e.Ch)
	}
	_, h := termbox.Size()
	printLine(1, h-1, string(b.prompt)+"  ", -1, -1)
	termbox.SetCursor(len(b.prompt)+1, h-1)
	if b.prompt == nil {
		termbox.HideCursor()
	}
}

func (b *BufferStack) handleCommand(cmd string, args []string, db *notmuch.Database) {
	switch cmd {
	case "close":
		b.Pop()
	case "quit":
		b.buffers = b.buffers[:0]
	case "search":
		if len(args) == 0 {
			b.Push(NewSearchBuffer("", db))
		} else {
			b.Push(NewSearchBuffer(args[0], db))
		}
	case "prompt":
		b.prompt = append(b.prompt, []rune(strings.Join(args, " ")+" ")...)
		if len(args) == 0 {
			b.prompt = b.prompt[:0]
		}
		_, h := termbox.Size()
		printLine(0, h-1, ":"+string(b.prompt), -1, -1)
		termbox.SetCursor(len(b.prompt)+1, h-1)
	}
}

func (b *BufferStack) HandleEvent(event *termbox.Event, db *notmuch.Database) {
	if len(b.buffers) == 0 {
		return
	}
	if event.Type == termbox.EventResize {
		b.refresh()
		return
	}

	if event.Type == termbox.EventKey {
		if b.prompt != nil {
			b.handlePrompt(event, db)
			return
		}
		cmd := getBinding("", event.Ch, event.Key)
		if cmd == nil {
			cmd := getBinding(b.buffers[len(b.buffers)-1].Name(), event.Ch, event.Key)
			if cmd == nil {
				return
			}
			b.buffers[len(b.buffers)-1].HandleCommand(cmd.Command, cmd.Args)
			return
		}
		b.handleCommand(cmd.Command, cmd.Args, db)
	}

}
