// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"strings"
)

type Part struct {
	Header textproto.MIMEHeader
	Body   string
}

type Mail struct {
	Header mail.Header
	Parts  []Part
}

// reads a mail and parses it into decoded parts
func readMail(filename string) (*Mail, error) {
	m := new(Mail)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	msg, err := mail.ReadMessage(file)
	if err != nil {
		return nil, err
	}

	m.Header = msg.Header

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	var boundary string
	if strings.HasPrefix(mediaType, "multipart/") {
		bodyReader = msg.Body
		boundary = params["boundary"]
	} else {
		// convert to multipart
		const boundaryText = "uaedt3rnc5trnu0aio94rane"
		buf := new(bytes.Buffer)
		buf.WriteString("--" + boundaryText + "\n")
		buf.WriteString("Content-Type: " + m.Header.Get("Content-Type") + "\n")
		buf.WriteString("Content-Transfer-Encoding: " + m.Header.Get("Content-Transfer-Encoding") + "\n\n")
		io.Copy(buf, msg.Body)
		buf.WriteString("\n--" + boundaryText + "--\n")

		bodyReader = buf
		boundary = boundaryText

	}
	mr := multipart.NewReader(bodyReader, boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		slurp, err := ioutil.ReadAll(p)
		if err != nil {
			return nil, err
		}
		m.Parts = append(m.Parts, Part{p.Header, string(slurp)})
	}

	return m, nil
}

func constructReply(m *Mail) *Mail {
	reply := new(Mail)
	reply.Header = make(mail.Header)

	reply.Header["MIME-Version"] = []string{"1.0"}
	reply.Header["Content-Transfer-Encoding"] = []string{"quoted-printable"}
	reply.Header["User-Agent"] = []string{"barely/0.1"}

	/*reply.Header["Content-Type"] = "text/plain; charset=\"utf-8\""*/
	return reply
}
