package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

var servers = map[string]string{}
var mut = sync.Mutex{}

func getServer(id string) string {
	mut.Lock()
	defer mut.Unlock()
	return servers[id]
}

func addServer(id, address string) {
	mut.Lock()
	defer mut.Unlock()
	servers[id] = address
}

func handleConnection(con net.Conn) {
	defer con.Close()

	r := bufio.NewReader(con)
	data, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("failed to read from connection %v, err: %v\n", con.RemoteAddr(), err)
		return
	}
	data = strings.Replace(data, "\n", "", 1)
	fmt.Printf("received: msg=%v from=%v\n", data, con.RemoteAddr())

	res := strings.Split(data, "=")
	if len(res) > 1 {
		remotePort := res[1]
		remoteAddr := con.RemoteAddr().String()[0:strings.LastIndex(con.RemoteAddr().String(), ":")]

		fmt.Printf("adding server: id=%v port=%v addr=%v\n", res[0], remotePort, remoteAddr)
		addServer(res[0], fmt.Sprintf("%v:%v", remoteAddr, remotePort))
		_, err = con.Write([]byte("ok"))
		if err != nil {
			fmt.Printf("failed to write to %v, err: %v\n", con.RemoteAddr(), err)
			return
		}
		return
	}

	fmt.Printf("requesting server %v\n", res[0])
	server := getServer(res[0])
	if server == "" { // no server
		fmt.Printf("server %v not found, writing response...\n", res[0])
		con.Write([]byte{0})
		return
	}
	fmt.Printf("found server %v with address %v, sending details to %v\n",
		res[0], server, con.RemoteAddr())
	_, err = con.Write([]byte(server))
	if err != nil {
		fmt.Printf("failed to send server details to %v, err: %v\n", con.RemoteAddr(), err)
	}
}

var addrFlag = flag.String("addr", "0.0.0.0:8080", "the address to listen on")

func main() {
	flag.Parse()
	addr := *addrFlag

	fmt.Printf("starting server on %v...\n", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("failed to start server, err: %v", err))
	}
	defer l.Close()
	fmt.Println("ready to accept connections")

	for {
		con, err := l.Accept()
		if err != nil {
			fmt.Printf("failed to accept connection, err: %v", err)
			continue
		}
		fmt.Printf("accepted connection from: %v\n", con.RemoteAddr())

		go handleConnection(con)
	}
}
