// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"net/textproto"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

// ComposeBuffer is a MailBuffer that allows editing and sending the viewed message.
type ComposeBuffer struct {
	mb *MailBuffer
}

// NewComposeBuffer creates a new Composebuffer for displaying a mail.
func NewComposeBuffer(m *Mail) *ComposeBuffer {
	return &ComposeBuffer{NewMailBufferFromMail(m)}
}

// Draw draws the buffer content.
func (b *ComposeBuffer) Draw() {
	b.mb.Draw()
}

// Title returns the buffer's title string.
func (b *ComposeBuffer) Title() string {
	return b.mb.mail.Header.Get("Message-ID")
}

// Name returns the buffer's name.
func (b *ComposeBuffer) Name() string {
	return "compose"
}

// Close closes the buffer.
func (b *ComposeBuffer) Close() {
	b.mb.Close()
}

// writeEditString writes an editable version of a mail consisting of a
// header paragraph containing some of the mails headers and the message body.
//
// The format is as follows:
//
//	Header1: value
//	Header2: value
//	Header3: value
//
//	Hi,
//	This is the message body text.
func writeEditString(filename string, m *Mail) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	file.Write([]byte("From: " + m.Header.Get("From") + "\n"))
	file.Write([]byte("To: " + m.Header.Get("To") + "\n"))
	file.Write([]byte("Subject: " + m.Header.Get("Subject") + "\n"))
	file.Write([]byte{'\n'})

	// assume the first part exists, is text/plain, and contains the message body.
	if len(m.Parts) == 0 {
		h := make(textproto.MIMEHeader)
		h["Content-Type"] = []string{"text/plain; charset=\"utf-8\""}
		h["Content-Transfer-Encoding"] = []string{"quoted-printable"}
		m.Parts = append(m.Parts, Part{h, ""})
	}
	file.Write([]byte(m.Parts[0].Body))
	return nil
}

// parseEditString parses a string created by writeEditString and edited by the user.
// it updates the changed header flags and message body.
func parseEditString(filename string, m *Mail) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var buf bytes.Buffer

	scanner := bufio.NewScanner(file)
	scanHeaders := true
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 && scanHeaders {
			scanHeaders = false // don't scan headers after new line.
			continue
		}
		if scanHeaders {
			toks := strings.SplitN(scanner.Text(), ":", 2)
			if len(toks) != 2 {
				return errors.New("Error: invalid header section.")
			}
			m.Header[strings.TrimSpace(toks[0])] =
				[]string{strings.TrimSpace(toks[1])}
		} else {
			_, err := buf.WriteString(scanner.Text() + "\n")
			if err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if len(m.Parts) == 0 {
		return errors.New("Error: editing invalid mail.")
	}
	m.Parts[0].Body = buf.String()
	return nil
}

func (b *ComposeBuffer) openEditor(stack *BufferStack) {
	filename := b.mb.tmpDir + "/edit.eml"

	err := writeEditString(filename, b.mb.mail)
	if err != nil {
		StatusLine = err.Error()
		return
	}

	termbox.Close()
	cmd := exec.Command(config.Commands.Editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		StatusLine = err.Error()
	}
	termbox.Init()
	err = parseEditString(filename, b.mb.mail)
	if err != nil {
		StatusLine = err.Error()
	}
	b.mb.mail.Header["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	b.mb.refreshBuf()
	stack.refresh()
}

// HandleCommand executes buffer local commands.
func (b *ComposeBuffer) HandleCommand(cmd string, args []string, stack *BufferStack) bool {
	switch cmd {
	case "reply", "raw": // disallow invalid commands in compose mode
	case "edit":
		b.openEditor(stack)
	case "send":
		StatusLine = "Sending..."
		stack.refresh()
		err := sendMail(b.mb.mail)
		if err != nil {
			StatusLine = err.Error()
		} else {
			StatusLine = "Mail sent."
		}
	case "attach":
		if len(args) == 0 {
			StatusLine = "Nothing to attach"
			break
		}

		err := b.mb.mail.attachFile(expandEnvHome(strings.Join(args, " ")))
		if err != nil {
			StatusLine = err.Error()
		} else {
			StatusLine = "attached \"" + strings.Join(args, " ") + "\""
		}
		b.mb.refreshBuf()
		b.Draw()
	case "deattach":
		if len(b.mb.mail.Parts) > 1 {
			b.mb.mail.Parts = b.mb.mail.Parts[:len(b.mb.mail.Parts)-1]
		}
		b.mb.refreshBuf()
		b.Draw()
	default:
		return b.mb.HandleCommand(cmd, args, stack)
	}
	return true
}
