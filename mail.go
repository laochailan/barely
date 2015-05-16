// Copyright 2015 Lukas Weber. All rights reserved.
// Use of this source code is governed by the MIT-styled
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"sort"
	"strings"
	"time"

	qp "gopkg.in/alexcesaro/quotedprintable.v2"
)

type Part struct {
	Header textproto.MIMEHeader
	Body   string
}

type Mail struct {
	Header mail.Header
	Parts  []Part
}

func readParts(reader io.Reader, boundary string, parts []Part) ([]Part, error) {
	mr := multipart.NewReader(reader, boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		mediaType, params, err := mime.ParseMediaType(p.Header.Get("Content-Type"))
		if strings.HasPrefix(mediaType, "multipart/") {
			parts, err = readParts(p, params["boundary"], parts)
			if err != nil {
				return nil, err
			}
		} else {
			r := io.Reader(p)
			if enc := p.Header.Get("Content-Transfer-Encoding"); enc == "base64" {
				r = base64.NewDecoder(base64.StdEncoding, p)
			}

			slurp, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			parts = append(parts, Part{p.Header, string(slurp)})
		}
	}
	return parts, nil
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

	m.Parts, err = readParts(bodyReader, boundary, m.Parts)

	return m, err
}

func composeMail() *Mail {
	m := new(Mail)
	m.Header = make(mail.Header)

	m.Header["MIME-Version"] = []string{"1.0"}
	m.Header["User-Agent"] = []string{UserAgend}

	t := time.Now()
	hostname, _ := os.Hostname()
	messageid := fmt.Sprintf("<%d%d%d%d%d.%x%x.%x@%s>", t.Year(), t.Month(),
		t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Unix(), hostname)
	m.Header["Message-ID"] = []string{messageid}
	m.Header["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	return m
}

func composeReply(m *Mail) *Mail {
	reply := composeMail()

	reply.Header["To"] = []string{m.Header.Get("From")}
	reply.Header["From"] = []string{m.Header.Get("To")}
	reply.Header["In-Reply-To"] = []string{m.Header.Get("Message-ID")}

	refs := m.Header["References"]

	newrefs := []string{}
	if len(refs) > 0 {
		newrefs = append(refs[:1], refs[max(1, len(refs)-8):]...)
	}
	newrefs = append(newrefs, m.Header.Get("Message-ID"))
	reply.Header["References"] = newrefs

	subj := m.Header.Get("Subject")
	if lower := strings.ToLower(subj); !strings.HasPrefix(lower, "re:") &&
		!strings.HasPrefix(lower, "aw:") {
		subj = "Re: " + subj
	}

	reply.Header["Subject"] = []string{subj}

	var replyBuf bytes.Buffer

	addr, err := m.Header.AddressList("From")
	name := ""
	if len(addr) != 0 {
		name = addr[0].Name
	}
	if name == "" || err != nil {
		name = m.Header.Get("From")
	}
	date, _ := m.Header.Date()
	datestr := date.Format("2006-01-02 15:04")

	replyBuf.WriteString(fmt.Sprintf("Quoting %s (%s):\n", name, datestr))
	for _, p := range m.Parts {
		contentType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		if contentType == "text/plain" {
			scanner := bufio.NewScanner(strings.NewReader(p.Body))
			for scanner.Scan() {
				replyBuf.WriteString("> ")
				replyBuf.WriteString(scanner.Text() + "\n")
			}
		}
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader["Content-Type"] = []string{"text/plain; charset=\"utf-8\""}
	partHeader["Content-Transfer-Encoding"] = []string{"quoted-printable"}
	reply.Parts = []Part{Part{partHeader, replyBuf.String()}}

	return reply
}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

func encodeMail(m *Mail) (string, error) {
	if len(m.Parts) == 0 {
		return "", errors.New("Error: message without content")
	}

	boundary := randomBoundary()

	if len(m.Parts) == 1 {
		for key, val := range m.Parts[0].Header {
			m.Header[key] = val
		}
	} else {
		m.Header["Content-Type"] = []string{"multipart/mixed; boundary=" + boundary}
	}

	var buffer bytes.Buffer
	henc := qp.Q.NewHeaderEncoder("utf-8")
	headers := make([]string, 0, len(m.Header))

	for key, val := range m.Header {
		s := strings.Join(val, " ")
		if s == "" {
			continue
		}
		if qp.NeedsEncoding(s) {
			s = henc.Encode(s)
		}
		headers = append(headers, key+": "+s+"\n")
	}
	sort.Strings(headers)
	for _, s := range headers {
		_, err := buffer.WriteString(s)
		if err != nil {
			return "", err
		}
	}
	_, err := buffer.WriteString("\n")
	if err != nil {
		return "", err
	}

	if len(m.Parts) == 1 {
		writer := qp.NewEncoder(&buffer)
		_, err := writer.Write([]byte(m.Parts[0].Body))
		if err != nil {
			return "", err
		}
	} else {
		mpw := multipart.NewWriter(&buffer)
		mpw.SetBoundary(boundary)
		for _, p := range m.Parts {
			pw, err := mpw.CreatePart(p.Header)
			if err != nil {
				return "", err
			}

			pw.Write([]byte(p.Body))
		}
		mpw.Close()
	}
	return buffer.String(), err
}
