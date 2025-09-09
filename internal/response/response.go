package response

import (
	"fmt"
	"io"

	"github.com/sevaergdm/httpfromtcp/internal/headers"
)

type writerStage int

const (
	stageStart writerStage = iota
	stageStatusWritten
	stageHeadersWritten
	stageBodyWriting
)

type Writer struct {
	dst          io.Writer
	stage        writerStage
}

type StatusCode int

const (
	OK                  StatusCode = 200
	BadRequest          StatusCode = 400
	InternalServerError StatusCode = 500
)

const crlf = "\r\n"

func NewWriter(dst io.Writer) *Writer {
	return &Writer{dst: dst, stage: stageStart}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.stage != stageStart {
		return fmt.Errorf("status line must be first; current stage=%v", w.stage)
	}

	var line string
	switch statusCode {
	case OK:
		line = "HTTP/1.1 200 OK\r\n"
	case BadRequest:
		line = "HTTP/1.1 400 Bad Request\r\n"
	case InternalServerError:
		line = "HTTP/1.1 500 Internal Server Error\r\n"
	default:
		line = fmt.Sprintf("HTTP/1.1 %d \r\n", statusCode)
	}
	_, err := w.dst.Write([]byte(line))
	if err != nil {
		return err
	}
	w.stage = stageStatusWritten
	return nil
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	header := headers.NewHeaders()
	header.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	header.Set("Connection", "close")
	header.Set("Content-Type", "text/plain")
	return header
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.stage != stageStatusWritten {
		return fmt.Errorf("headers must be after status line; current stage=%v", w.stage)
	}
	for k, v := range headers {
		_, err := w.dst.Write([]byte(fmt.Sprintf("%s: %s\r\n", k, v)))
		if err != nil {
			return err
		}
	}
	_, err := w.dst.Write([]byte("\r\n"))
	w.stage = stageHeadersWritten
	return err
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.stage != stageHeadersWritten && w.stage != stageBodyWriting {
		return 0, fmt.Errorf("body must be after headers; current stage=%v", w.stage)
	}
	n, err := w.dst.Write(p)
	if err == nil {
		w.stage = stageBodyWriting
	}
	return n, err
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	chunkSize := fmt.Sprintf("%x", len(p))

	_, err :=	w.dst.Write([]byte(chunkSize))
	if err != nil {
		return 0, err
	}

	_, err = w.dst.Write([]byte(crlf))
	if err != nil {
		return 0, err
	}

	n, err := w.dst.Write(p)
	if err != nil {
		return n, err
	}

	_, err = w.dst.Write([]byte(crlf))
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	totalBytes := 0
	n, err := w.dst.Write([]byte("0"))
	if err != nil {
		return 0, err
	}
	totalBytes += n

	n, err = w.dst.Write([]byte(crlf))
	if err != nil {
		return 0, err
	}
	totalBytes += n

	n, err = w.dst.Write([]byte(crlf))
	if err != nil {
		return 0, err
	}
	totalBytes += n

	return totalBytes, nil 
}
