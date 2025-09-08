package response

import (
	"fmt"
	"io"

	"github.com/sevaergdm/httpfromtcp/internal/headers"
)

type StatusCode int

const (
	OK StatusCode = 200
	BadRequest StatusCode = 400
	InternalServerError StatusCode = 500
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	switch statusCode {
	case OK:
		_, err := w.Write([]byte("HTTP/1.1 200 OK\r\n"))
		return err
	case BadRequest:
		_, err := w.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		return err
	case InternalServerError:
		_, err := w.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n"))
		return err
	default:
		_, err := w.Write([]byte(fmt.Sprintf("HTTP/1.1 %d \r\n", statusCode)))
		return err

	}
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	header := headers.NewHeaders()
	header.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	header.Set("Connection", "close")
	header.Set("Content-Type", "text/plain")
	return header
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for k, v := range headers {
		_, err := w.Write([]byte(fmt.Sprintf("%s: %s\r\n", k, v)))
		if err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("\r\n"))
	return err
}
