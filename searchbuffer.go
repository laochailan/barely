// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"time"
	"unicode/utf8"

	"github.com/laochailan/notmuch/bindings/go/src/notmuch"
	termbox "github.com/nsf/termbox-go"
)

// Results is a general list iterator behaving like notmuch.Threads or notmuch.Messages
type results interface {
	Valid() bool
	Get() result
	MoveToNext()
	Destroy()
}

type result interface {
	GetSubject() string
	GetDate() int64
	GetAuthor() string
	GetTags() *notmuch.Tags
}

type threadResults struct {
	*notmuch.Threads
}

func (tr *threadResults) Get() result {
	return &threadResult{tr.Threads.Get()}
}

type threadResult struct {
	*notmuch.Thread
}

func (t *threadResult) GetDate() int64 {
	return t.GetNewestDate()
}

func (t *threadResult) GetAuthor() string {
	return t.GetAuthors()
}

type messageResults struct {
	*notmuch.Messages
}

func (mr *messageResults) Get() result {
	return &messageResult{mr.Messages.Get()}
}

type messageResult struct {
	*notmuch.Message
}

func (m *messageResult) GetDate() int64 {
	date, _ := m.Message.GetDate()
	return date
}

func (m *messageResult) GetAuthor() string {
	return m.GetHeader("From")
}

func (m *messageResult) GetSubject() string {
	return m.GetHeader("Subject")
}

// SearchBuffer can be used to display a list of threads matching
// a specific search term.
type SearchBuffer struct {
	term string // Search term
	typ  SearchType

	database *notmuch.Database
	messages []result
	msgit    results
	query    *notmuch.Query

	cursor int
}

// SearchType is the type of the objects searched for.
type SearchType int

// Possible values for SearchType
const (
	STMessages SearchType = iota
	STThreads
)

// NewSearchBuffer creates a new Searchbuffer for a search term of type typ.
func NewSearchBuffer(term string, typ SearchType) *SearchBuffer {
	var status notmuch.Status
	buf := new(SearchBuffer)
	buf.term = term
	buf.typ = typ

	buf.database, status = notmuch.OpenDatabase(expandEnvHome(config.General.Database), 0)
	if status != notmuch.STATUS_SUCCESS {
		StatusLine = status.String()
	}

	buf.refreshQuery()
	return buf
}

func tagString(msg result) (strs []string, fgs []int) {
	if msg == nil {
		panic("looked for tags of nil msg")
	}

	strs = make([]string, 0, 4)
	fgs = make([]int, len(strs))
	tags := msg.GetTags()

	for tags.Valid() {
		tag := tags.Get()

		tagFg := config.Theme.Tags
		if color, exists := pconfig.TagColors[tag]; exists {
			tagFg = color
		}
		if alias, exists := pconfig.TagAliases[tag]; exists {
			tag = alias
		}
		if tag != "" {
			strs = append(strs, tag)
			fgs = append(fgs, tagFg)
		}
		tags.MoveToNext()
	}

	return strs, fgs
}

// Draw draws the content of the buffer.
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

		t := b.messages[i+offset].GetDate()
		date := shortTime(time.Unix(t, 0))
		from := shortFrom(b.messages[i+offset].GetAuthor())
		subj := b.messages[i+offset].GetSubject()

		tags, tagFgs := tagString(b.messages[i+offset])

		dateFg := config.Theme.Date
		fromFg := config.Theme.From
		subjFg := config.Theme.Subject

		// Do not color line if we are under the cursor
		if i+offset == b.cursor {
			dateFg = -1
			fromFg = -1
			subjFg = -1
		}
		printLine(1, i, date, dateFg, -1)

		tagLength := 0
		for j := range tags {
			if i+offset == b.cursor {
				tagFgs[j] = -1
			}
			printLine(10+tagLength, i, tags[j], tagFgs[j], -1)
			tagLength += utf8.RuneCountInString(tags[j]) + 1
		}
		printLine(11+tagLength-1, i, from, fromFg, -1)
		printLine(12+len(from)+tagLength, i, subj, subjFg, -1)

	}
}

// Title returns the title string of the buffer.
func (b *SearchBuffer) Title() string {
	msg := ""
	if b.typ == STMessages {
		msg = "messages "
	}
	return msg + "for \"" + b.term + "\""
}

// Name returns the name of the buffer.
func (b *SearchBuffer) Name() string {
	return "search"
}

// Close closes the buffer.
func (b *SearchBuffer) Close() {
	b.query.Destroy()
	b.database.Close()
}

// tagCmd is used to manipulate tags of the message.
// cmd can be either "tag" or "untag"
func (b *SearchBuffer) tagCmd(cmd string, tags []string) error {
	if len(b.messages) == 0 {
		return errors.New("No messages to tag")
	}
	db, status := notmuch.OpenDatabase(expandEnvHome(config.General.Database), 1)
	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}
	defer db.Close()

	queryStr := ""
	if b.typ == STMessages {
		queryStr = "id:" + b.messages[b.cursor].(*messageResult).GetMessageId()
	} else {
		queryStr = "thread:" + b.messages[b.cursor].(*threadResult).GetThreadId()
	}

	query := db.CreateQuery(queryStr)
	msgit := query.SearchMessages()
	if msgit == nil {
		return errors.New("Message not found")
	}

	for msgit.Valid() {
		msg := msgit.Get()
		msg.Freeze()

		for _, tag := range tags {
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
		if config.General.Synchronize_Flags {
			status = msg.TagsToMaildirFlags()
			if status != 0 {
				return errors.New(status.String())
			}
		}
		msgit.MoveToNext()
	}
	query.Destroy()

	return nil
}

// refreshQuery reopens the database connection and refreshes the search.
func (b *SearchBuffer) refreshQuery() {
	var status notmuch.Status
	if b.query != nil {
		b.query.Destroy()
	}
	b.database.Close()
	b.database, status = notmuch.OpenDatabase(expandEnvHome(config.General.Database), 0)
	if status != notmuch.STATUS_SUCCESS {
		StatusLine = status.String()
		return
	}

	_, h := termbox.Size()
	b.messages = make([]result, 0, h)
	b.query = b.database.CreateQuery(b.term)

	if b.typ == STMessages {
		it := b.query.SearchMessages()
		if it == nil {
			b.msgit = nil
		} else {
			b.msgit = &messageResults{it}
		}
	} else {
		it := b.query.SearchThreads()
		if it == nil {
			b.msgit = nil
		} else {
			b.msgit = &threadResults{it}
		}
	}

	if b.msgit == nil {
		StatusLine = "Could not refresh buffer"
		return
	}

	for i := 0; i <= b.cursor && b.msgit.Valid(); i++ {
		b.messages = append(b.messages, b.msgit.Get())
		b.msgit.MoveToNext()
	}

	if b.cursor >= len(b.messages) {
		b.cursor = max(0, len(b.messages)-1)
	}
}

// HandleCommand handles buffer local commands.
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
		if b.typ == STThreads { // open a list of messages in the thread instead
			threadid := b.messages[b.cursor].(*threadResult).GetThreadId()
			stack.Push(NewSearchBuffer("thread:"+threadid, STMessages))
			break
		} else if b.typ == STMessages {
			if b.cursor >= 0 && b.cursor < len(b.messages) {
				stack.Push(NewMailBuffer(b.messages[b.cursor].(*messageResult).GetFileName()))
			}
		}
	case "tag", "untag":
		err := b.tagCmd(cmd, args)
		if err != nil {
			StatusLine = err.Error()
		}
		b.refreshQuery()
		b.Draw()
	case "_refresh":
		b.refreshQuery()
	default:
		return false
	}
	return true
}
