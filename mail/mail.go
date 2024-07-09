package mail

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Mail struct {
	// Imap server address with port eg: outlook.office365.com:993
	Addr string
	// User email address
	User string
	// User email password,
	// For gmail it will be app password not regular
	// For outlook it will be regular password
	Pass string
	// Connection object to the imap server
	con *client.Client
	// Inbox folder connection
	ibox *imap.MailboxStatus
	// Total number of messages in the INBOX folder
	numMsgs uint32
}

// Establishes the connection with given imap server
// Logsin the user with given email and password
// Selects the INBOX folder
func (m *Mail) Login() error {
	log.Println("Connecting to server...")

	conf := &tls.Config{
		Rand: rand.Reader,
	}

	// Connect to server
	var err error
	m.con, err = client.DialTLS(m.Addr, conf)
	if err != nil {
		return fmt.Errorf("unable to connect to %v. err: %v", m.Addr, err.Error())
	}
	log.Println("Connected")

	// Login
	if err := m.con.Login(m.User, m.Pass); err != nil {
		return fmt.Errorf("unable to login to %v. err: %v", m.User, err.Error())
	}
	log.Println("Logged in")

	// Select Inbox
	m.ibox, err = m.con.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("unable to read INBOX. err: %v", err.Error())
	}
	m.numMsgs = m.ibox.Messages
	return nil
}

// Logs the user out
// Closes the connection with the imap server
func (m *Mail) Logout() {
	m.con.Logout()
	fmt.Println("Loggedout")
}

// Fetches messages for a given range
func (m *Mail) Fetch(from, to uint32) []Message {

	msgs := make([]Message, to-from+1)

	// Make seqset
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		// Fetch the entire body with attachments
		// Also fetch the message envolope
		done <- m.con.Fetch(seqset, []imap.FetchItem{imap.FetchItem("BODY.PEEK[]"), imap.FetchEnvelope}, messages)
	}()

	idx := 0
	for msg := range messages {
		// Grab the message Id
		msgs[idx].Id = msg.Envelope.MessageId

		// For each body section
		for _, literal := range msg.Body {
			// Parse the Message segments
			msgs[idx].parse(literal)
		}
		idx++
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	// Sort messages based on date, latest first
	sortMsgs(msgs)

	return msgs
}

// Calls the Fetch method until a message with t(date) found
func (m *Mail) FetchAfter(t time.Time) []Message {

	since := time.Since(t)
	found := false
	// Prepare to and from
	to := m.numMsgs
	// Fetch 10 recent msgs
	from := to - 9

	if to < 10 {
		from = uint32(1)
	}

	msgs := make([]Message, 0, to-from+1)

	// Loop until a message with t(date) is found
	for !found && from > 0 && to > 0 {
		if to < 10 {
			from = 1
		}
		// Call Fetch method
		for _, msg := range m.Fetch(from, to) {
			// If we find a message with date <= t we stop fetching
			if time.Since(msg.Date) > since {
				found = true
				break
			}
			msgs = append(msgs, msg)
		}

		// Prepare to and from for the next 10 messages
		to = from - 1
		from = to - 9

	}

	// Sort messages based on date, latest first
	sortMsgs(msgs)

	return msgs
}

// Returns total number of messages in the INBOX folder
func (m *Mail) NumMsgs() uint32 {
	return m.numMsgs
}

// Sorts messages based on date, latest first
func sortMsgs(msgs []Message) {
	sort.Slice(msgs, func(i, j int) bool {
		return time.Since(msgs[i].Date) < time.Since(msgs[j].Date)
	})
}
