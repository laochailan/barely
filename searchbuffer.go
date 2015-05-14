// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/notmuch/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

// SearchBuffer can be used to display a list of threads matching
// a specific search term.
type SearchBuffer struct {
	term     string // Search term
	database *notmuch.Database
	messages []*notmuch.Message
	msgit    *notmuch.Messages
	query    *notmuch.Query

	cursor int
}

func NewSearchBuffer(term string, db *notmuch.Database) *SearchBuffer {
	buf := new(SearchBuffer)
	buf.term = term
	buf.database = db

	_, h := termbox.Size()
	buf.messages = make([]*notmuch.Message, 0, h)
	buf.query = db.CreateQuery(term)
	buf.msgit = buf.query.SearchMessages()

	for i := 0; i < h && buf.msgit.Valid(); i++ {
		buf.messages = append(buf.messages, buf.msgit.Get())
		buf.msgit.MoveToNext()
	}
	return buf
}

func (b *SearchBuffer) Draw() {
	w, h := termbox.Size()
	cbuf := termbox.CellBuffer()

	offset := 0
	if b.cursor >= h*3/4 {
		offset = -h*3/4 + b.cursor
	}

	for b.msgit.Valid() && len(b.messages) < h-2+offset {
		for i := 0; i < 20 && b.msgit.Valid(); i++ {
			b.messages = append(b.messages, b.msgit.Get())
			b.msgit.MoveToNext()
		}
	}
	for i := 0; i < h-2; i++ {
		for x := 0; x < w; x++ {
			cbuf[i*w+x].Ch = 0
			if i+offset == b.cursor {
				cbuf[i*w+x].Fg = termbox.Attribute(config.Theme.HlFg) |
					termbox.AttrBold
				cbuf[i*w+x].Bg = termbox.Attribute(config.Theme.HlBg)
			} else {
				cbuf[i*w+x].Fg = 0
				cbuf[i*w+x].Bg = 0
			}
		}

		if i+offset < 0 || i+offset >= len(b.messages) {
			continue
		}

		t, _ := b.messages[i+offset].GetDate()
		date := shortTime(time.Unix(t, 0))
		from := shortFrom(b.messages[i+offset].GetHeader("From"))
		subj := b.messages[i+offset].GetHeader("Subject")

		if i+offset == b.cursor {
			printLine(1, i, date, -1, -1)
			printLine(10, i, from, -1, -1)
			printLine(11+len(from), i, subj, -1, -1)
		} else {
			printLine(1, i, date, config.Theme.Date, -1)
			printLine(10, i, from, config.Theme.From, -1)
			printLine(11+len(from), i, subj, config.Theme.Subject, -1)
		}
	}
}

func (b *SearchBuffer) Title() string {
	return "for \"" + b.term + "\""
}

func (b *SearchBuffer) Name() string {
	return "search"
}

func (b *SearchBuffer) Close() {
	b.query.Destroy()
}

func (b *SearchBuffer) HandleCommand(cmd string, args []string, stack *BufferStack) bool {
	switch cmd {
	case "move":
		if len(args) == 0 {
			break
		}
		_, h := termbox.Size()
		switch args[0] {
		case "up":
			b.cursor--
		case "down":
			b.cursor++
		case "pageup":
			b.cursor -= h
		case "pagedown":
			b.cursor += h
		}
		if b.cursor < 0 {
			b.cursor = 0
		}
		b.Draw()
		if !b.msgit.Valid() && b.cursor >= len(b.messages) {
			b.cursor = len(b.messages) - 1
			b.Draw()
		}
	case "show":
		stack.Push(NewMailBuffer(b.messages[b.cursor].GetFileName()))
	default:
		return false
	}
	return true
}
