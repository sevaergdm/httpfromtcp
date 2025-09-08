package main

import (
	"fmt"
	"log"
	"net"

	"github.com/sevaergdm/httpfromtcp/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		
		go func(c net.Conn){
			fmt.Println("Connection Accepted!")
			
			req, err := request.RequestFromReader(conn)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Request line:")
			fmt.Printf("- Method: %s\n", req.RequestLine.Method)
			fmt.Printf("- Target: %s\n", req.RequestLine.RequestTarget)
			fmt.Printf("- Version: %s\n", req.RequestLine.HttpVersion)
			fmt.Println("Headers:")
			for k, v := range req.Headers  {
				fmt.Printf("- %s: %s\n", k, v)
			}
			fmt.Println("Body:")
			fmt.Println(string(req.Body))

			fmt.Println("Connection has been closed")
		}(conn)
	}
}

