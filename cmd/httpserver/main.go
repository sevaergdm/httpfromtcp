package main

import (
	"crypto/sha256"
	"encoding/hex"
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
	server, err := server.Serve(routingHandler, port)
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

func routingHandler(w *response.Writer, r *request.Request) {
	if strings.HasPrefix(r.RequestLine.RequestTarget, "/httpbin/") {
		proxyHandler(w, r)
		return
	}else if strings.HasPrefix(r.RequestLine.RequestTarget, "/video") {
		handlerVideo(w, r)
		return
	} else {
		handler(w, r)
		return
	}
}

func proxyHandler(w *response.Writer, r *request.Request) {
	target := strings.TrimPrefix(r.RequestLine.RequestTarget, "/httpbin/")
	url := "https://httpbin.org/" + target
	log.Printf("Proxying to: %s\n", url)
	header := headers.NewHeaders()

	resp, err := http.Get(url)
	log.Printf("Got response status code: %d", resp.StatusCode)
	if err != nil {
		log.Printf("Unable to make get request: %v", err)
		w.WriteStatusLine(response.StatusCode(http.StatusBadGateway))
		w.WriteHeaders(header)
		return
	}
	defer resp.Body.Close()

	w.WriteStatusLine(response.StatusCode(resp.StatusCode))
	for k, v := range resp.Header {
		if k == "Content-Length" {
			continue
		}
		vString := strings.Join(v, ", ")
		header.Set(k, vString)
	}
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Trailer", "X-Content-SHA256, X-Content-Length")
	w.WriteHeaders(header)

	buf := make([]byte, 1024)
	var body []byte
	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("All bytes read")
				break
			}
			log.Printf("Error while reading response body: %v", err)
			break
		}
		if n == 0 {
			continue
		}
		log.Printf("Successfully read %d bytes", n)

		_, err = w.WriteChunkedBody(buf[:n])
		if err != nil {
			log.Println("Unable to write chunk")
			break
		}
		body = append(body, buf[:n]...)
		log.Printf("Successfully wrote %d bytes", n)
		log.Printf("Body is %d bytes", len(body))
	}

	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		log.Printf("Unable to write closing to buffer: %v", err)
	}

	trailers := headers.NewHeaders()
	sum := sha256.Sum256(body)
	trailers.Set("X-Content-SHA256", hex.EncodeToString(sum[:]))
	trailers.Set("X-Content-Length", fmt.Sprintf("%d", len(body)))
	err = w.WriteTrailers(trailers)
	if err != nil {
		log.Println("Unable to write trailers")
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

func handlerVideo(w *response.Writer, r *request.Request) {
	w.WriteStatusLine(response.OK)
	header := headers.NewHeaders()
	header.Set("Content-Type", "video/mp4")
	w.WriteHeaders(header)

	body, err := os.ReadFile("../../assets/vim.mp4")
	if err != nil {
		log.Printf("Unable to read video: %v", err)
		return
	}

	_, err = w.WriteBody(body)
	if err != nil {
		log.Printf("Unable to write video: %v", err)
		return
	}
}
