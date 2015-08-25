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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/djimenez/iconv-go"
	"github.com/laochailan/barely/maildir"
	"github.com/laochailan/notmuch/bindings/go/src/notmuch"
	"github.com/saintfish/chardet"
	qp "gopkg.in/alexcesaro/quotedprintable.v2"
)

// Part represents a multipart part. Messages that do not have multipart content
// are still represented as multipart messages with one part internally.
type Part struct {
	Header textproto.MIMEHeader
	Body   string
}

// Mail represents the content of one mail message.
type Mail struct {
	Header mail.Header
	Parts  []Part
}

// readParts read parts out of a multipart body (including nested multiparts).
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
			slurp = convertToUtf8(slurp)

			parts = append(parts, Part{p.Header, string(slurp)})
		}
	}
	return parts, nil
}

// convertToUtf8 dectects the charset of the given plain text slice and converts it to utf-8
// if necessary
func convertToUtf8(text []byte) (converted []byte) {
	detector := chardet.NewTextDetector()
	res, err := detector.DetectBest(text)
	if err != nil { // give up on error
		return text
	}

	if res.Charset == "UTF-8" { // nothing to do
		return text
	}

	conv, err := iconv.NewConverter(res.Charset, "UTF-8")
	if err != nil {
		return text
	}
	defer conv.Close()
	convStr, err := conv.ConvertString(string(text))
	if err != nil {
		return text
	}
	return []byte(convStr)
}

// readMail reads a mail and parses it into decoded parts
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
		mediaType = "text/plain"
		params = make(map[string]string)
		params["charset"] = "utf-8"
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

// composeMail creates a new Mail structure from scratch.
func composeMail() *Mail {
	m := new(Mail)
	m.Header = make(mail.Header)

	m.Header["MIME-Version"] = []string{"1.0"}
	m.Header["User-Agent"] = []string{UserAgent}

	t := time.Now()
	hostname, _ := os.Hostname()
	messageid := fmt.Sprintf("<%d%d%d%d%d.%x%x.%x@%s>", t.Year(), t.Month(),
		t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Unix(), hostname)
	m.Header["Message-ID"] = []string{messageid}
	m.Header["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	return m
}

// composeReply creates a Mail structure for a Reply to Mail m.
func composeReply(m *Mail) *Mail {
	reply := composeMail()

	to, _, err := qp.DecodeHeader(m.Header.Get("From"))
	if err != nil {
		to = m.Header.Get("From")
	}
	from, _, err := qp.DecodeHeader(m.Header.Get("To"))
	if err != nil {
		from = m.Header.Get("To")
	}
	reply.Header["To"] = []string{to}
	reply.Header["From"] = []string{from}
	reply.Header["In-Reply-To"] = []string{m.Header.Get("Message-ID")}

	refs := m.Header["References"]

	newrefs := []string{}
	if len(refs) > 0 {
		newrefs = append(refs[:1], refs[max(1, len(refs)-8):]...)
	}
	newrefs = append(newrefs, m.Header.Get("Message-ID"))
	reply.Header["References"] = newrefs

	subj, _, err := qp.DecodeHeader(m.Header.Get("Subject"))
	if err != nil {
		subj = m.Header.Get("Subject")
	}
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
	reply.Parts = []Part{{partHeader, replyBuf.String()}}

	return reply
}

// randomBoundary creates a boundary to be used for multipart e-mail bodies.
// It was taken from the mime/multipart package.
func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

// attachMail adds a file as an attachment to the mail.
func (m *Mail) attachFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	nlInsert := newNewlineInserter(&buf, 78)
	enc := base64.NewEncoder(base64.StdEncoding, nlInsert)
	_, err = io.Copy(enc, file)
	if err != nil {
		return err
	}
	enc.Close()
	typ := mime.TypeByExtension(filepath.Ext(filename))
	name := filepath.Base(filename)
	if typ == "" {
		typ = "application/octet-stream"
	}

	header := make(textproto.MIMEHeader)
	header["Content-Type"] = []string{typ + "; name=\"" + name + "\""}
	header["Content-Disposition"] = []string{"attachment; filename=\"" + name + "\""}
	header["Content-Transfer-Encoding"] = []string{"base64"}
	m.Parts = append(m.Parts, Part{header, buf.String()})

	return nil
}

// Encode encodes a Mail structure to 7bit text.
func (m *Mail) Encode() (string, error) {
	if len(m.Parts) == 0 {
		return "", errors.New("Error: message without content")
	}

	henc := qp.Q.NewHeaderEncoder("utf-8")
	boundary := randomBoundary()

	if len(m.Parts) == 1 {
		for key, val := range m.Parts[0].Header {
			m.Header[key] = val
		}
	} else {
		m.Header["Content-Type"] = []string{"multipart/mixed; boundary=" + boundary}
	}

	var buffer bytes.Buffer
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
			writer := pw
			if p.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
				writer = qp.NewEncoder(pw)
			}

			writer.Write([]byte(p.Body))
		}
		mpw.Close()
	}
	return buffer.String(), err
}

func sendMail(m *Mail) error {

	addrl, err := m.Header.AddressList("From")
	if len(addrl) != 1 {
		return errors.New("Invalid count of addresses in 'From' field.")
	}

	addr := addrl[0]

	account := getAccount(addr.Address)
	if account == nil {
		return fmt.Errorf("No account configured for '%s'", addr.Address)
	}

	switch {
	case account.Sent_Dir == "":
		return errors.New("No sent-dir configured for account.")
	case account.Sendmail_Command == "":
		return errors.New("No sendmail-command configured for account.")
	}

	mailcont, err := m.Encode()
	if err != nil {
		return err
	}

	strcmd := strings.Split(account.Sendmail_Command, " ")
	cmd := exec.Command(strcmd[0], strcmd[1:]...)
	cmd.Stdin = strings.NewReader(mailcont)
	output, _ := cmd.CombinedOutput()
	if len(output) != 0 {
		return errors.New(string(output))
	}

	filename, err := maildir.Store(os.ExpandEnv(account.Sent_Dir), []byte(mailcont), "S")
	if err != nil {
		return err
	}

	db, status := notmuch.OpenDatabase(os.ExpandEnv(config.General.Database), 1)
	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}
	defer db.Close()

	msg, status := db.AddMessage(filename)
	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}
	defer msg.Destroy()

	msg.Freeze()
	defer msg.Thaw()
	msg.RemoveAllTags()
	for _, tag := range account.Sent_Tag {
		status = msg.AddTag(tag)
	}
	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}

	// TODO: maybe make this cleaner.
	if len(m.Parts) > 1 {
		status = msg.AddTag("attachment")
	}

	if status != notmuch.STATUS_SUCCESS {
		return errors.New(status.String())
	}

	return nil
}

// inserts a newline character after lineLength bytes. only for ascii because in wider
// encoding runes could be chopped up.
type newlineInserter struct {
	w          io.Writer
	lineLength int
	counter    int
}

func newNewlineInserter(w io.Writer, lineLength int) *newlineInserter {
	return &newlineInserter{w, lineLength, 0}
}

func (n *newlineInserter) Write(data []byte) (int, error) {
	written := 0
	for n.counter+len(data) > n.lineLength {
		num, err := n.w.Write(data[:n.lineLength-n.counter])
		written += num
		if err != nil {
			return written, err
		}
		num, err = n.w.Write([]byte{'\n'})
		written += num
		if err != nil {
			return written, err
		}

		data = data[n.lineLength-n.counter:]
		n.counter = 0
	}

	num, err := n.w.Write(data)
	written += num
	n.counter += num
	return written, err
}
