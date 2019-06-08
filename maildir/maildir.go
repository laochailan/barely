// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

// Package maildir provides methods for interacting with the Maildir format as specified at http://cr.yp.to/proto/maildir.html
package maildir

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrorUnknownKey = errors.New("Key does not exist in Maildir.")
var ErrorDuplicateKey = errors.New("Key exists more than once.")

func key() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	var buf [10]byte
	_, err = io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		return "", err
	}

	t := time.Now()
	return fmt.Sprintf("%v.M%vP%vR%x.%v", t.Unix(), t.Nanosecond()/1000, os.Getpid(), string(buf[:]), hostname), nil
}

func splitKeyFlags(filename string) (key, flags string, err error) {
	splits := strings.Split(filepath.Base(filename), ":2,")
	if len(splits) != 2 {
		return "", "", errors.New("Mail filename format violation.")
	}

	return splits[0], splits[1], nil
}

func joinKeyFlags(key, flags string) string {
	return key + ":2," + flags
}

// Dir represents a single Maildir folder.
type Dir struct {
	path string
}

// Open opens a maildir. Given create, if the "cur", "tmp", or "new" directories do not exist, they are created.
func Open(path string, create bool) (md *Dir, err error) {
	md = &Dir{path}
	if !create {
		return md, nil
	}

	err = os.Mkdir(path, os.ModeDir|0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("creating Maildir failed: %s", err)
	}

	err = os.Mkdir(filepath.Join(path, "new"), os.ModeDir|0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("creating Maildir failed: %s", err)
	}

	err = os.Mkdir(filepath.Join(path, "cur"), os.ModeDir|0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("creating Maildir failed: %s", err)
	}

	err = os.Mkdir(filepath.Join(path, "tmp"), os.ModeDir|0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("creating Maildir failed: %s", err)
	}

	return md, nil
}

// NewMessage writes a raw mail into the Maildir. After writing is complete, the corresponding file is moved to the "new" directory.
func (md *Dir) NewMessage(mail []byte) (msg *Message, err error) {
	msg = new(Message)

	msg.md = md
	msg.Key, err = key()
	if err != nil {
		return nil, err
	}

	tmpname := filepath.Join(md.path, "tmp", msg.Key)
	file, err := os.Create(tmpname)
	if err != nil {
		return nil, err
	}

	size, err := file.Write(mail)
	file.Close()
	if err != nil {
		os.Remove(tmpname)
		return nil, err
	}

	name := fmt.Sprintf("%s,S=%d", msg.Key, size)

	target := "new"

	msg.filename = filepath.Join(md.path, target, name)
	err = os.Rename(tmpname, msg.filename)
	if err != nil {
		os.Remove(tmpname)
		return nil, err
	}

	return msg, nil
}

func (md *Dir) keyToFilename(key string) (filename string, err error) {
	matches, err := filepath.Glob(filepath.Join(md.path, "[new,cur]", key+"*"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", ErrorUnknownKey
	}
	if len(matches) > 1 {
		return "", ErrorDuplicateKey
	}

	filename = matches[0]
	return
}

// Get finds a message in the Maildir by its unique Key.
func (md *Dir) Get(key string) (msg *Message, err error) {
	fn, err := md.keyToFilename(key)
	msg = &Message{key, fn, md}
	return
}

// List returns a list of all messages in the Maildir
func (md *Dir) List() (msgs []Message, err error) {
	filenames, err := filepath.Glob(filepath.Join(md.path, "[new,cur]", "*"))
	if err != nil {
		return nil, err
	}

	msgs = make([]Message, len(filenames))

	for i, f := range filenames {
		msgs[i].Key, _, err = splitKeyFlags(f)
		if err != nil {
			return nil, err
		}

		msgs[i].filename = f
		msgs[i].md = md
	}

	return msgs, nil
}

// Message represents a file in a Maildir.
type Message struct {
	// Key is a unique string identifying the message in the Maildir
	Key string

	// last known filename. May be invalid.
	filename string

	// Maildir the mail belongs to
	md *Dir
}

// Flags returns the current flags stored in the message filename.
func (msg *Message) Flags() (flags string, err error) {
	fn, err := msg.Filename()
	if err != nil {
		return "", err
	}

	_, flags, err = splitKeyFlags(fn)
	return
}

// Filename returns the current path of the file containing Message.
// Treat with caution as it may become invalid after flags are changed by another MUA for example.
func (msg *Message) Filename() (string, error) {
	_, err := os.Stat(msg.filename)
	if err == nil {
		return msg.filename, nil
	}

	return msg.md.keyToFilename(msg.Key)
}

// SetFlags sets the flags on a message and moves it into the "cur" directory (if not already).
// newFlags is a string containing at most one letter of the flags defined by the specification: PRSTDF.
func (msg *Message) SetFlags(newFlags string) error {
	sortedFlags := []byte(newFlags)
	sort.Slice(sortedFlags, func(i, j int) bool { return sortedFlags[i] < sortedFlags[j] })

	filename, err := msg.Filename()
	if err != nil {
		return err
	}

	newFilename := filepath.Join(msg.md.path, "cur", joinKeyFlags(msg.Key, string(sortedFlags)))
	err = os.Rename(filename, newFilename)
	msg.filename = newFilename

	return err
}
