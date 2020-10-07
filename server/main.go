package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
)

const (
	defaultNegotiatorAddr = "localhost:8080"
	maxBuf                = 256
)

func getPort(addr net.Addr) string {
	parts := strings.Split(addr.String(), ":")
	if len(parts) < 2 {
		panic(fmt.Errorf("no ports found on addr: %v", addr))
	}

	return parts[len(parts)-1]
}

func sendDetailsToNegotiator(id, negoAddr string, addr net.Addr) error {
	buf := make([]byte, 128)
	con, err := net.Dial("tcp", negoAddr)
	if err != nil {
		return err
	}
	defer con.Close()

	localPort := getPort(addr)
	_, err = con.Write([]byte(fmt.Sprintf("%v=%v\n", id, localPort)))
	if err != nil {
		return err
	}
	fmt.Println("sent details to negotiator, awaiting confirmation...")

	n, err := con.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	fmt.Printf("got response: %v\n", string(buf[0:n]))

	return nil
}

func handleConnection(con net.Conn) {
	defer con.Close()
	buf := make([]byte, maxBuf)

	for {
		n, err := con.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("%v disconnected\n", con.RemoteAddr())
				return
			}
			fmt.Printf("failed to read from %v, err: %v\n", con.RemoteAddr(), err)
			continue
		}
		fmt.Printf("[received:] addr=%v msg=%v\n", con.RemoteAddr(), string(buf[0:n]))
		_, err = con.Write(buf[0:n])
		if err != nil {
			fmt.Printf("failed to write to %v, err: %v\n", con.RemoteAddr(), err)
			continue
		}
	}
}

var negoAddrFlag = flag.String("negotiator-addr", defaultNegotiatorAddr, "the address of the negotiator server")
var idFlag = flag.String("id", "0", "id of the server")

func main() {
	flag.Parse()
	negoAddr := *negoAddrFlag
	id := *idFlag

	// start server
	fmt.Println("staring echo server...")
	l, err := net.Listen("tcp", "")
	if err != nil {
		panic(fmt.Errorf("failed to start tcp server, err: %v", err))
	}
	defer l.Close()
	fmt.Printf("started listening on: %v\n", l.Addr())

	// send details to negotiator
	fmt.Printf("sending details to negotiator service on %v\n", negoAddr)
	err = sendDetailsToNegotiator(id, negoAddr, l.Addr())
	if err != nil {
		panic(fmt.Errorf("failed to send details to negotiator server, err: %v", err))
	}

	fmt.Println("registered, waiting for incoming connections...")
	for {
		con, err := l.Accept()
		if err != nil {
			fmt.Printf("failed to accept tcp connection, err: %v\n", err)
		}
		fmt.Printf("accepted connection from: %v\n", con.RemoteAddr())
		go handleConnection(con)
	}
}
