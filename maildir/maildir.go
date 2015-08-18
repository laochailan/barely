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
	return fmt.Sprintf("%v.M%vP%vR%x.%v", t.Unix(), t.Nanosecond()/1000, os.Getpid(), string(buf[:]), hostname), nil
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
