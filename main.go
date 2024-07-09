package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/tars47/go-read-mail/awss3"
	"github.com/tars47/go-read-mail/excel"
	"github.com/tars47/go-read-mail/mail"
)

// Default name for the excel file
const DefaultExcel = "data.xlsx"

func main() {

	// This handles the request
	http.HandleFunc("POST /", readMail)

	// Start the server
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// Response struct that will be sent to the user
type response struct {
	Status   int    `json:"status"`
	Message  string `json:"message"`
	ExcelUrl string `json:"excelUrl"`
}

// Handler function that process the user request
// Checks if user email and excel file is already present
// If present reads the lastest message and fetches new messages
// If not present fetches lastest 25 messages and saves it to s3
func readMail(w http.ResponseWriter, r *http.Request) {

	var u mail.Mail
	// Decode the user request and validate
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil || u.Addr == "" || u.User == "" || u.Pass == "" {
		send(w, response{Status: http.StatusBadRequest, Message: "Malformed request body"})
		return
	}

	// Connect to the imap address provides
	// Login the user with user email and password provided
	// Select the INBOX folder
	if err := u.Login(); err != nil {
		send(w, response{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}
	defer u.Logout()

	// Check s3 if user email folder already exists format: example@gmail.com/data.xlsx
	// If present returns bytes buffer
	s3buf, err := awss3.DownloadFile(fmt.Sprintf("%s/%s", u.User, DefaultExcel))
	if err != nil {
		// If file not present we assume this is a new user
		if strings.Contains(err.Error(), awss3.NotFound) {
			// Fetches recent 25 messages from imap server and creates excel file and uploads to s3
			// returns the s3 presigned url
			url, err := createUserExcel(&u)
			if err != nil {
				send(w, response{Status: http.StatusInternalServerError, Message: err.Error()})
				return
			}
			// Sends the response back to client, response containes excel s3 url
			send(w, response{Status: http.StatusCreated, Message: "Success", ExcelUrl: url})
			return
		}
		send(w, response{Status: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// Reads the recent message from the buffer and fetches all messages after the recent message
	// Updates the excel
	// Replaces the s3 file
	// Returns presigned s3 url
	url, err := updateUserExcel(&u, s3buf)
	if err != nil {
		send(w, response{Status: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// Sends the response back to client, response containes excel s3 url
	send(w, response{Status: http.StatusCreated, Message: "Success", ExcelUrl: url})
}

// Fetches recent 25 messages
// Uploads all the attachments to s3 concurrently
// Creates new excel file
// Uploads the excel file to s3
// Returns the presigned s3 url
func createUserExcel(u *mail.Mail) (string, error) {
	// Get total messages in the INBOX folder
	to := u.NumMsgs()
	from := to - 25

	if to <= 25 {
		from = uint32(1)
	}
	// Fetches recent 25 messages
	msgs := u.Fetch(from, to)

	// Uploads all the attachments to s3 concurrently
	uploadAttachments(u, msgs)

	// Creates new excel file
	ebuf, err := excel.New(msgs)
	if err != nil {
		return "", fmt.Errorf("unable to create excel file. err: %s", err.Error())
	}
	// Uploads the excel file to s3
	url, err := awss3.UploadFile(fmt.Sprintf("%s/%s", u.User, DefaultExcel), ebuf)
	if err != nil {
		return "", fmt.Errorf("unable to upload excel file. err: %s", err.Error())
	}
	// Returns the presigned s3 url
	return url, nil
}

// Fetches recent record message date in excel
// Fetches all messages after the last message date
// Uploads all the attachments to s3 concurrently
// Prepends the excel with the newly fetched messages
// Replaces the s3 file
// Returns the presigned s3 url
func updateUserExcel(u *mail.Mail, buf *bytes.Buffer) (string, error) {
	// Duplicate the buf received from s3
	var bufc bytes.Buffer
	tee := io.TeeReader(buf, &bufc)

	// Reads the recent message date
	t := excel.GetRecentMsgDate(tee)

	// Fetches all the messages after recent message date
	msgs := u.FetchAfter(t)
	// If no messages found generate the presigned url and return
	if len(msgs) == 0 {
		return awss3.GetFileLink(fmt.Sprintf("%s/%s", u.User, DefaultExcel))
	}

	// Uploads all the attachments to s3 concurrently
	uploadAttachments(u, msgs)

	// Prepends the excel with the newly fetched messages
	bufp, err := excel.PrependRows(&bufc, msgs)
	if err != nil {
		return "", fmt.Errorf("unable to update excel file. err: %s", err.Error())
	}
	// Replaces the s3 file and get pre signed s3 url
	url, err := awss3.UploadFile(fmt.Sprintf("%s/%s", u.User, DefaultExcel), bufp)
	if err != nil {
		return "", fmt.Errorf("unable to upload excel file. err: %s", err.Error())
	}
	// Return pre signed s3 url
	return url, nil

}

// Launches a go routine to upload to s3 concurrently
func uploadAttachments(u *mail.Mail, msgs []mail.Message) {
	var wg sync.WaitGroup
	for _, msg := range msgs {
		for i := 0; i < len(msg.Attachment); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				url, err := awss3.UploadFile(fmt.Sprintf("%s/%s/%s", u.User, msg.Id, msg.Attachment[i].Name), &msg.Attachment[i].Buf)
				if err != nil {
					fmt.Printf("[uploadAttachments] err uploading attachment %s, user: %s. err: %s\n", msg.Attachment[i].Name, u.User, err.Error())
					return
				}
				msg.Attachment[i].Url = url
			}()
		}
	}

	wg.Wait()
}

// Helper function that sends the response back to client
func send(w http.ResponseWriter, res response) {
	w.Header().Set("Content-Type", "application/json")

	bytes, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{status:500,message:InternalServerError}"))
		return
	}

	w.WriteHeader(res.Status)
	w.Write([]byte(bytes))
}
