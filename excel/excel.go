package excel

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/tars47/go-read-mail/mail"
	"github.com/xuri/excelize/v2"
)

// Default sheet name
var s1 = "Sheet1"

// Headers in the excel file
var headers = []string{"Id", "Date", "From", "Subject", "Cc", "Bcc", "ReplyTo", "Attachments"}

// Creates a new excel file
// Writes Headers ansd given message rows
// Writes and returns the data to a bytes.Buffer
func New(msgs []mail.Message) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	setHeaders(f)

	setRows(f, msgs)

	return save(f)
}

// Reads the excel data from the given reader
// Grabs the B2 cell value (recent message date)
// Parses date and returns time.Time
func GetRecentMsgDate(r io.Reader) time.Time {
	// Read from r
	f, err := excelize.OpenReader(r)
	if err != nil {
		log.Printf("[GetRecentMsgDate] err reading: %v\n", err)
		return time.Time{}
	}
	defer f.Close()

	// Grab the B2 cell value(recent message date)
	dtstr, err := f.GetCellValue(s1, "B2")
	if err != nil {
		log.Printf("[GetRecentMsgDate] err getting B2 cell value: %v\n", err)
		return time.Time{}
	}

	// Parse the time to a format
	t, err := time.Parse("2006-01-02 15:04:05 -0700", dtstr)
	if err != nil {
		log.Printf("[GetRecentMsgDate] err parsing time: %v\n", err)
		return time.Time{}
	}
	// Returns time.Time
	return t
}

// Prepends the messages rows to the data read from r
func PrependRows(r io.Reader, msgs []mail.Message) (*bytes.Buffer, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		log.Printf("[PrependRows] err reading: %v\n", err)
		return nil, err
	}
	defer f.Close()

	err = f.InsertRows(s1, 2, len(msgs))
	if err != nil {
		log.Printf("[PrependRows] err inserting rows: %v\n", err)
		return nil, err
	}

	setRows(f, msgs)

	return save(f)
}

// Writes the default headers
func setHeaders(f *excelize.File) {
	// Header style
	style, _ := f.NewStyle(
		&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Font:      &excelize.Font{Bold: true, Color: "#000080"},
		})

	for i, header := range headers {
		col := fmt.Sprint(string(rune(65 + i)))
		cell := fmt.Sprintf("%s%d", col, 1)

		f.SetCellValue(s1, cell, header)
		f.SetCellStyle(s1, cell, cell, style)

		switch header {
		case "Id":
			f.SetColWidth(s1, col, col, 100)
		case "Date":
			f.SetColWidth(s1, col, col, 30)
		case "From":
			f.SetColWidth(s1, col, col, 50)
		case "Subject":
			f.SetColWidth(s1, col, col, 100)
		default:
			f.SetColWidth(s1, col, col, 50)
		}
	}
}

// Writes the message rows
func setRows(f *excelize.File, msgs []mail.Message) {
	// default style
	style, _ := f.NewStyle(
		&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
		},
	)
	// attachments url style
	linkStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Font:      &excelize.Font{Color: "#1265BE", Underline: "single"},
	})

	f.SetRowHeight(s1, 1, 15)

	for i, msg := range msgs {

		dataRow := i + 2
		f.SetRowHeight(s1, dataRow, 25)

		for j, h := range headers {

			colRune := rune(65 + j)
			cell := fmt.Sprintf("%s%d", string(colRune), dataRow)
			f.SetCellStyle(s1, cell, cell, style)

			switch h {
			case "Id":
				f.SetCellValue(s1, cell, msg.Id)
			case "Date":
				f.SetCellValue(s1, cell, msg.Date.Format("2006-01-02 15:04:05 -0700"))
			case "From":
				f.SetCellValue(s1, cell, mail.ToString(msg.From))
			case "Subject":
				f.SetCellValue(s1, cell, msg.Subject)
			case "Cc":
				f.SetCellValue(s1, cell, mail.ToString(msg.Cc))
			case "Bcc":
				f.SetCellValue(s1, cell, mail.ToString(msg.Bcc))
			case "ReplyTo":
				f.SetCellValue(s1, cell, mail.ToString(msg.ReplyTo))
			case "Attachments":
				// Loop each attachment and set the hyperlink
				// If multiple attachments are present,
				// I am storing each attachment in new cells statting from H column
				// As I could not figure out a way to write comma seperated links in one cell
				for k, att := range msg.Attachment {
					acell := fmt.Sprintf("%s%d", string(rune(65+j+k)), dataRow)
					f.SetCellHyperLink(s1, acell, att.Url, "External")
					f.SetCellValue(s1, acell, att.Name)
					f.SetCellStyle(s1, acell, acell, linkStyle)
				}

			}
		}
	}
}

// Writes to buffer and returns pointer to bytes buffer
func save(f *excelize.File) (*bytes.Buffer, error) {
	// Write to a buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		log.Printf("[save] err writing to buffer: %v\n", err)
		return nil, err
	}

	return buf, nil
}
