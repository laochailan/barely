package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNewlineInserter(t *testing.T) {
	var buf bytes.Buffer
	ni := newNewlineInserter(&buf, 4)
	ni.Write([]byte("aaaa"))
	ni.Write([]byte("aaaaaaaaaa"))
	fmt.Println(buf.String())
	if buf.String() != "aaaa\naaaa\naaaa\naa" {
		t.Fail()
	}
}
