package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	// define flags for host and port
	host := flag.String("host", "localhost", "the host to connect to")
	port := flag.String("port", "27017", "the port to connect to")
	timeout := 10 * time.Second

	// parse the flags
	flag.Parse()

	for i := 1; i < 20; i++ {
		// try to connect to the server
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(*host, *port), timeout)
		if err == nil {
			// connection successful, close it and exit the loop
			conn.Close()
			fmt.Printf("TCP connection available on [%s:%s]\n", *host, *port)
			return
		}

		// connection unsuccessful, print error and retry after a short delay
		fmt.Printf("connection not yet available on [%s:%s]: %v\n", *host, *port, err)
		time.Sleep(1 * time.Second)
	}
	log.Panicf("could not open TCP connection on [%s,%s] after max attempts.", *host, *port)
}
