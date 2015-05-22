// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"os"
	"strings"
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

	buf.refreshQuery()
	return buf
}

func tagString(msg *notmuch.Message) string {
	if msg == nil {
		panic("looked for tags of nil msg")
	}

	strs := make([]string, 0, 4)
	tags := msg.GetTags()

	for tags.Valid() {
		tag := tags.Get()
		if alias, exists := pconfig.TagAliases[tag]; exists {
			tag = alias
		}
		if tag != "" {
			strs = append(strs, tag)
		}
		tags.MoveToNext()
	}
	return strings.Join(strs, " ")
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

		tags := tagString(b.messages[i+offset])

		dateFg := config.Theme.Date
		fromFg := config.Theme.From
		subjFg := config.Theme.Subject
		tagsFg := config.Theme.Tags
		if i+offset == b.cursor {
			dateFg = -1
			fromFg = -1
			subjFg = -1
			tagsFg = -1
		}
		printLine(1, i, date, dateFg, -1)
		printLine(10, i, tags, tagsFg, -1)
		printLine(11+len(tags), i, from, fromFg, -1)
		printLine(12+len(from)+len(tags), i, subj, subjFg, -1)

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

func (b *SearchBuffer) tagCmd(cmd string, args []string) error {
	if len(b.messages) == 0 {
		return errors.New("No messages to tag")
	}
	db, status := notmuch.OpenDatabase(os.ExpandEnv(config.General.Database), 1)
	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}
	defer db.Close()

	msg, status := db.FindMessage(b.messages[b.cursor].GetMessageId())
	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}
	msg.Freeze()

	for _, tag := range args {
		switch cmd {
		case "tag":
			status = msg.AddTag(tag)
		case "untag":
			status = msg.RemoveTag(tag)
		}

		if status != 0 {
			return errors.New(status.String())
		}
	}
	status = msg.Thaw()
	if status != 0 {
		return errors.New(status.String())
	}
	return nil
}

func (b *SearchBuffer) refreshQuery() {
	if b.query != nil {
		b.query.Destroy()
	}
	_, h := termbox.Size()
	b.messages = make([]*notmuch.Message, 0, h)
	b.query = b.database.CreateQuery(b.term)
	b.msgit = b.query.SearchMessages()

	for i := 0; i < h && b.msgit.Valid(); i++ {
		b.messages = append(b.messages, b.msgit.Get())
		b.msgit.MoveToNext()
	}
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
			b.cursor = max(0, len(b.messages)-1)
			b.Draw()
		}
	case "show":
		if b.cursor >= 0 && b.cursor < len(b.messages) {
			stack.Push(NewMailBuffer(b.messages[b.cursor].GetFileName()))
		}
	case "tag", "untag":
		err := b.tagCmd(cmd, args)
		if err != nil {
			StatusLine = err.Error()
		}
	case "refresh":
		b.refreshQuery()
	default:
		return false
	}
	return true
}
