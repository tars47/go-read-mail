package mail

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
)

var addressList = []string{"From", "Sender", "Cc", "Bcc", "Reply-To"}

type Message struct {
	Id string

	Date     time.Time
	Subject  string
	BodyText string
	BodyHtml string

	Cc      []string
	Bcc     []string
	From    []string
	Sender  []string
	ReplyTo []string

	Attachment []Attachment
}

type Attachment struct {
	Name string
	Type string
	Url  string
	Buf  bytes.Buffer
}

// Reads the message segments
// Parses all the fields in Message struct
func (m *Message) parse(l io.Reader) {

	var err error

	// Create a mail reader
	mr, err := mail.CreateReader(l)
	if err != nil {
		log.Printf("failed to create mail reader: %v\n", err)
		return
	}

	// Parse header fields
	h := mr.Header

	// Grab the message Date
	if m.Date, err = h.Date(); err != nil {
		log.Printf("failed to parse Date header field: %v\n", err)
	}
	// Grab the message Subject
	if m.Subject, err = h.Text("Subject"); err != nil {
		log.Printf("failed to parse Subject header field: %v\n", err)
	}

	// Parse "From", "Sender", "Cc", "Bcc", "Reply-To"
	for _, field := range addressList {
		fval, err := h.AddressList(field)
		if err != nil {
			log.Printf("failed to parse To %v header field: %v\n", field, err)
			continue
		}

		si := make([]string, 0, len(fval))
		for _, v := range fval {
			si = append(si, v.String())
		}

		switch field {
		case "From":
			m.From = si
		case "Sender":
			m.Sender = si
		case "Cc":
			m.Cc = si
		case "Bcc":
			m.Bcc = si
		case "Reply-To":
			m.ReplyTo = si
		}
	}

	// Process the body's parts
	for {

		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("failed to read message part: %v\n", err)
			return
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			b, _ := io.ReadAll(p.Body)
			ctype, _, _ := h.ContentType()
			switch ctype {
			case "text/plain":
				m.BodyText = string(b)
			case "text/html":
				m.BodyHtml = string(b)
			}

		case *mail.AttachmentHeader:
			// This is an attachment
			name, _ := h.Filename()
			ctype, _, _ := h.ContentType()
			b, _ := io.ReadAll(p.Body)

			// Write the attachment bytes to a buffer
			var buf bytes.Buffer
			buf.Write(b)

			m.Attachment = append(m.Attachment, Attachment{Name: name, Type: ctype, Buf: buf})
		}
	}

	// Convert date to utc
	m.Date = m.Date.UTC()
	// Trim spaces
	m.Subject = strings.TrimSpace(m.Subject)
	m.BodyText = strings.TrimSpace(m.BodyText)
	m.BodyHtml = strings.TrimSpace(m.BodyHtml)
}

// Method that converts a Message struct into human readable format
func (m *Message) String() {
	fmt.Println("******************************************************************")
	fmt.Printf("Id:\t%v\n", m.Id)

	fmt.Printf("From:\t%v\n", ToString(m.From))
	fmt.Printf("CC:\t%v\n", ToString(m.Cc))
	fmt.Printf("BCC:\t%v\n", ToString(m.Bcc))
	fmt.Printf("Sender:\t%v\n", ToString(m.Sender))
	fmt.Printf("ReplyTo:\t%v\n", ToString(m.ReplyTo))

	fmt.Printf("Date:\t%v\n", m.Date)
	fmt.Printf("Subject:\t%v\n", m.Subject)

	if len(m.BodyText) > 50 {
		fmt.Printf("BodyText:\t%v\n", m.BodyText[:50]+"...")
	} else {
		fmt.Printf("BodyText:\t%v\n", m.BodyText)
	}

	if len(m.BodyHtml) > 50 {
		fmt.Printf("BodyHtml:\t%v\n", m.BodyHtml[:50]+"...")
	} else {
		fmt.Printf("BodyHtml:\t%v\n", m.BodyHtml)
	}

	for i, att := range m.Attachment {
		fmt.Printf("Attachment(%v):\n", i)
		att.String()
	}
}

// Method that converts a Attachment struct into human readable format
func (a *Attachment) String() {
	fmt.Printf("\tName: %v\n", a.Name)
	fmt.Printf("\tType: %v\n", a.Type)
	fmt.Printf("\tSize: %vkb\n", len(a.Buf.Bytes())/1000)
}

// Converts slice of strings to comma sepecated value string
func ToString(si []string) string {
	var buf bytes.Buffer
	for i := 0; i < len(si); i++ {
		buf.WriteString(si[i])
		if i < len(si)-1 {
			buf.WriteString(", ")
		}
	}
	return buf.String()
}
