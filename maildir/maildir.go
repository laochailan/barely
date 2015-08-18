// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

// Package maildir provides simple routines to add messages to a Maildir.
package maildir

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	return fmt.Sprintf("%v.M%vP%vR%v.%v", t.Unix(), t.Nanosecond()/1000, os.Getpid(), buf, hostname), nil
}

// Store stores a message an a given maildir.
// flags must be given as a sorted string of valid Maildir flags, e.g. "FRS".
//
// If flags is empty, the mail is stored in maidir/new. If not, it’s stored to maildir/cur.
func Store(maildir string, mail []byte, flags string) (filename string, err error) {
	_, err = os.Stat(maildir)
	if err != nil {
		return "", err
	}

	basename, err := key()
	if err != nil {
		return "", err
	}
	tmpname := filepath.Join(maildir, "tmp", basename)
	file, err := os.Create(tmpname)
	if err != nil {
		return "", err
	}

	size, err := file.Write(mail)
	file.Close()
	if err != nil {
		os.Remove(tmpname)
		return "", err
	}

	name := fmt.Sprintf("%s,S=%d", basename, size)

	target := "new"
	if len(flags) != 0 {
		target = "cur"
		name += ":2," + flags
	}

	filename = filepath.Join(maildir, target, name)
	err = os.Rename(tmpname, filename)
	if err != nil {
		os.Remove(tmpname)
		return "", err
	}

	return filename, nil
}

func flagName(path string, flag byte, on bool) (string, error) {
	idx := strings.LastIndex(path, ":2,")
	if idx == -1 {
		return "", fmt.Errorf("'%s' does not contain maildir flags", path)
	}

	flagstr := path[idx+3:]
	if i := strings.IndexByte(flagstr, flag); i != -1 {
		if !on {
			flagstr = flagstr[:i] + flagstr[i+1:]
		}
	} else if on {
		i = strings.IndexFunc(flagstr, func(r rune) bool { return r > rune(flag) })
		if i == -1 {
			i = 0
		}
		flagstr = flagstr[:i] + string(flag) + flagstr[i:]
	}

	newName := path[:idx+3] + flagstr
	return newName, nil
}

// Flag turns a maildir flag on or off on the mail stored in path.
func Flag(path string, flag byte, on bool) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	newPath, err := flagName(path, flag, on)
	if err != nil {
		return err
	}
	err = os.Rename(path, newPath)
	return err
}
