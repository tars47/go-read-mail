# Go-Read-Mail

Implements functionality to

1. Connect to a given imap server
2. Pull user email messages
3. Writes email messages to an excel file
4. Uploads email message attachments and excel file to s3
5. Generates s3 pre signed url to the user excel file

## Curl request

```
curl --location 'localhost:3000/' \
--header 'Content-Type: application/json' \
--data-raw '{
    "addr": "outlook.office365.com:993",
    "user": "xxxx@outlook.com",
    "pass": "xxxxxxxxxxx"
}'

response:
{
    "status": 201,
    "message": "Success",
    "excelUrl": "https://xxx.s3.xx-xxx-x.amazonaws.com/xxx%40outlook.com/data.xlsx?X-Amz-Algorithm=xxx&X-Amz-Credential=xxx&X-Amz-Date=xxx&X-Amz-Expires=604800&X-Amz-SignedHeaders=xxx&x-id=GetObject&X-Amz-Signature=xx"
}
```

## Mail Package

```go
// Represents a Mail Message type
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

// Represents a Mail Message Attachment type
type Attachment struct {
	Name string
	Type string
	Url  string
	Buf  bytes.Buffer
}

user := mail.Mail{
          Addr: "outlook.office365.com:993",  // for gmail use "imap.gmail.com:993"
          User: "emailId",
          Pass: "password"                    // for gmail, regular password will not work,
                                              // generate app password
        }

// Login the user and selects INBOX folder
if err := user.Login(); err != nil {
	// err handling
}
defer user.Logout()

// Fetch messages in range
to := user.NumMsgs() // total messages present in INBOX folder
from := to - 25

if to <= 25 {
	from = uint32(1)
}
// Fetches recent 25 messages
msgs := user.Fetch(from, to) // both from and to is of type uint32 and returns []mail.Message

// Fetch recent messages after a given time
t, _ := time.Parse("2006-01-02 15:04:05 -0700", "2024-07-01 00:00:00 +0000")

msgs := user.FetchAfter(t) // takes in time.Time and returns []mail.Message

```

## Excel Package

```go
// Creates new excel file
buf, err := excel.New(msgs) // takes in []mail.Message and returns *bytes.Buffer,error
if err != nil {
	// err handling
}

// Reads the recent message date (cell value of B2)
t := excel.GetRecentMsgDate(reader) //takes in io.Reader and return time.Time

// Prepends the excel with the newly fetched messages
buf, err := excel.PrependRows(&bufc, msgs) // takes in *bytes.Buffer and []mail.Message
                                           // returns *bytes.Buffer
if err != nil {
	// err handling
}
```

## Awss3 Package

```go
// Upload to s3
url, err := awss3.UploadFile(key, buf) // takes object key and io.Reader, returns presigned url
if err != nil {
	// err handling
}

// Download file
buf, err := awss3.DownloadFile(key) // takes object key and returns *bytes.Buffer
if err != nil {
	// err handling
}

// Get presigned url
url,err := awss3.GetFileLink(key) // takes object key and returns presigned url
if err != nil {
	// err handling
}
```
