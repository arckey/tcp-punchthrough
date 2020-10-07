package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

const (
	defaultNegotiatorAddr = "localhost:8080"
	maxBuf                = 256
)

var negoAddrFlag = flag.String("negotiator-addr", defaultNegotiatorAddr, "the address of the negotiator server")

func main() {
	flag.Parse()
	negoAddr := *negoAddrFlag

	buf := make([]byte, maxBuf)

	// connect to negotiator server
	fmt.Printf("connecting to negotiator on: %v...\n", negoAddr)
	con, err := net.Dial("tcp", negoAddr)
	if err != nil {
		panic(fmt.Errorf("failed to connect to negotiator server on %v, err: %v", negoAddr, err))
	}

	fmt.Println("connected to negotiator server")

	// request target server details
	fmt.Print("enter target server id: ")
	n, err := os.Stdin.Read(buf)
	if err != nil {
		panic(fmt.Errorf("could not get target server id, err: %v", err))
	}
	if n == 0 || buf[0] == '\n' {
		panic(fmt.Errorf("must enter id"))
	}

	n, err = con.Write(buf[0:n])
	if err != nil {
		panic(fmt.Errorf("failed to send target server id, err: %v", err))
	}

	fmt.Println("waiting for target server details...")
	n, err = con.Read(buf)
	if err != nil && err != io.EOF {
		panic(fmt.Errorf("failed to get target server details, err: %v", err))
	}
	if n < 2 {
		panic(fmt.Errorf("target server not found"))
	}
	con.Close()

	// attempt to connect to the target server
	targetAddr := string(buf[0:n])
	fmt.Printf("received target server address: %v\n", targetAddr)
	fmt.Println("attempting to connect...")

	con, err = net.Dial("tcp", targetAddr)
	if err != nil {
		panic(fmt.Errorf("failed to connect to target server, err: %v", err))
	}
	defer con.Close()

	fmt.Printf("connected to: %v\n", targetAddr)
	for {
		fmt.Printf("[msg:] ")
		n, err = os.Stdin.Read(buf)
		if err != nil {
			panic(fmt.Errorf("failed to read message, err: %v", err))
		}
		n, err = con.Write(buf[0:n])
		if err != nil {
			panic(fmt.Errorf("failed to write message, err: %v", err))
		}
		n, err = con.Read(buf)
		if err != nil {
			panic(fmt.Errorf("failed to read response from target server, err: %v", err))
		}
		fmt.Printf("[resp:] %v", string(buf[0:n]))
	}
}
