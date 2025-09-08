package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/sevaergdm/httpfromtcp/internal/headers"
)

type Request struct {
	RequestLine RequestLine
	State       requestState
	Headers     headers.Headers
	Body        []byte
	bodyLengthRead int
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type requestState int

const (
	requestStateInitialized requestState = iota
	requestStateDone
	requestStateParsingHeaders
	requestStateParseBody
)

const crlf = "\r\n"
const bufferSize = 8

var ErrNeedMoreData = errors.New("need more data")

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize)
	readToIndex := 0

	request := &Request{
		State:   requestStateInitialized,
		Headers: headers.NewHeaders(),
		Body: make([]byte, 0),
	}

	for request.State != requestStateDone {
		if readToIndex >= len(buf) {
			newBuf := make([]byte, 2*len(buf))
			copy(newBuf, buf)
			buf = newBuf
		}

		numBytesRead, err := reader.Read(buf[readToIndex:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				if request.State != requestStateDone {
					return nil, fmt.Errorf("incomplete request, in state: %d, read n bytes on EOF: %d", request.State, numBytesRead)
				}
				break
			}
			return nil, err
		}
		readToIndex += numBytesRead

		numBytesParsed, err := request.parse(buf[:readToIndex])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[numBytesParsed:])
		readToIndex -= numBytesParsed
	}

	return request, nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	idx := bytes.Index(data, []byte(crlf))
	if idx == -1 {
		return nil, 0, nil
	}

	requestLineText := string(data[:idx])
	requestLine, err := requestLineFromString(requestLineText)
	if err != nil {
		return nil, 0, err
	}

	return requestLine, idx + 2, nil
}

func requestLineFromString(str string) (*RequestLine, error) {
	parts := strings.Split(str, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid requestline: %s", str)
	}

	method := parts[0]
	var isUpperLetter = regexp.MustCompile(`^[A-Z]+$`).MatchString
	if !isUpperLetter(method) {
		return nil, fmt.Errorf("invalid method: %s", method)
	}

	requestTarget := parts[1]

	versionParts := strings.Split(parts[2], "/")
	if len(versionParts) != 2 {
		return nil, fmt.Errorf("malformed start-line: %s", str)
	}

	httpPart := versionParts[0]
	if httpPart != "HTTP" {
		return nil, fmt.Errorf("Unrecognized http version: %s", httpPart)
	}

	version := versionParts[1]
	if version != "1.1" {
		return nil, fmt.Errorf("Unrecognized http version: %s", version)
	}

	return &RequestLine{
		Method:        method,
		RequestTarget: requestTarget,
		HttpVersion:   version,
	}, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	for r.State != requestStateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		totalBytesParsed += n
		if n == 0 {
			break
		}
	}
	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.State {
	case requestStateInitialized:
		requestLine, numBytes, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}

		if numBytes == 0 {
			return 0, nil
		}

		r.RequestLine = *requestLine
		r.State = requestStateParsingHeaders
		return numBytes, nil
	case requestStateParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}

		if done {
			r.State = requestStateParseBody
		}
		return n, nil
	case requestStateParseBody:
		contentLenStr, ok := r.Headers.Get("Content-Length")
		if !ok {
			r.State = requestStateDone
			return len(data), nil
		}
		contentLen, err := strconv.Atoi(contentLenStr)
		if err != nil {
			return 0, fmt.Errorf("malformed Content-Length: %s", err)
		}

		r.Body = append(r.Body, data...)
		r.bodyLengthRead += len(data)
		if  r.bodyLengthRead > contentLen {
			return 0, fmt.Errorf("Content-Length too large")
		}
		if r.bodyLengthRead == contentLen {
			r.State = requestStateDone
		}
		return len(data), nil
	case requestStateDone:
		return 0, fmt.Errorf("error: trying to read data in a done state")
	default:
		return 0, fmt.Errorf("error: unknown state")
	}
}
