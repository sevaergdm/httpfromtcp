package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sevaergdm/httpfromtcp/internal/headers"
	"github.com/sevaergdm/httpfromtcp/internal/request"
	"github.com/sevaergdm/httpfromtcp/internal/response"
	"github.com/sevaergdm/httpfromtcp/internal/server"
)

const port = 42069

func main() {
	server, err := server.Serve(handler, port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

func proxyHandler(w *response.Writer, r *request.Request) {
	proxyPrefix := "/httpbin/"
	proxyTarget := "https://httpbin.org/"
	revisedTarget := ""
	header := headers.NewHeaders()
	if strings.HasPrefix(r.RequestLine.RequestTarget, proxyPrefix) {
		path := strings.TrimPrefix(r.RequestLine.RequestTarget, proxyPrefix)
		revisedTarget = proxyTarget + path
	} else {
		log.Printf("Incorrect proxy path sent: %s", r.RequestLine.RequestTarget)
		w.WriteStatusLine(response.BadRequest)
		return
	}

	resp, err := http.Get(revisedTarget)
	if err != nil {
		log.Printf("Unable to make get request: %v", err)
		w.WriteStatusLine(http.StatusBadGateway)
		w.WriteHeaders(header)
		return
	}

	w.WriteStatusLine(response.StatusCode(resp.StatusCode))
	for k, v := range resp.Header {
		if k == "Content-Length" {
			continue
		}
		vString := strings.Join(v, ", ")
		header.Set(k, vString)
	}
	header.Set("Transfer-Encoding", "chunked")
	w.WriteHeaders(header)

	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("All bytes read")
				break
			}
		}
		fmt.Printf("Successfully read %d bytes", n)

		n, err = w.WriteChunkedBody(buf[:n])
		if err != nil {
			fmt.Println("Unable to write chunk")
		}
		fmt.Printf("Successfully wrote %d bytes", n)	
	}

	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		fmt.Printf("Unable to write closing to buffer: %d", err)
	}
}

func handler(w *response.Writer, r *request.Request) {
	if r.RequestLine.RequestTarget == "/yourproblem" {
		w.WriteStatusLine(response.BadRequest)
		header := headers.NewHeaders()
		body := []byte("<html><head><title>400 Bad Request</title></head><body><h1>Bad Request</h1><p>Your request honestly kinda sucked.</p></body></html>")

		header.Set("Content-Type", "text/html")
		header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeaders(header)
		w.WriteBody(body)
		return
	}

	if r.RequestLine.RequestTarget == "/myproblem" {
		w.WriteStatusLine(response.InternalServerError)
		header := headers.NewHeaders()
		body := []byte(`<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`)

		header.Set("Content-Type", "text/html")
		header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeaders(header)
		w.WriteBody(body)
		return
	}

	w.WriteStatusLine(response.OK)
	header := headers.NewHeaders()
	body := []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`)

	header.Set("Content-Type", "text/html")
	header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeaders(header)
	w.WriteBody(body)
	return
}
