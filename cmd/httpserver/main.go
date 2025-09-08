package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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
