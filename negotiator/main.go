package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/arckey/tcp-punchthrough/helpers"
	"github.com/arckey/tcp-punchthrough/types/peer"
	"github.com/arckey/tcp-punchthrough/types/request"
	fb "github.com/google/flatbuffers/go"
)

// maps a name of a peer to local and remote address
var servers = map[string]struct {
	peer *peer.Peer
	con  net.Conn
}{}
var mut = sync.Mutex{}

func getPeer(id string) (*peer.Peer, net.Conn, bool) {
	mut.Lock()
	defer mut.Unlock()
	p, ok := servers[id]
	return p.peer, p.con, ok
}

func addPeer(id string, p *peer.Peer, con net.Conn) {
	mut.Lock()
	defer mut.Unlock()
	servers[id] = struct {
		peer *peer.Peer
		con  net.Conn
	}{
		peer: p,
		con:  con,
	}
}

func handleConnection(con net.Conn) {
	buf := make([]byte, 512)

	for {
		n, err := con.Read(buf)
		if err == io.EOF {
			fmt.Printf("connection closed with %v\n", con.RemoteAddr())
			con.Close()
			return
		}
		helpers.PanicIfErr("cannot read from connection", err)

		req := request.GetRootAsRequest(buf[:n], 0)
		reqTable := &fb.Table{}
		req.Request(reqTable)

		switch req.Type() {
		case request.RequestTypeRegistration:
			rr := &request.RegistrationRequest{}
			rr.Init(reqTable.Bytes, reqTable.Pos)
			handleRegistrationReq(con, rr)
		case request.RequestTypeConnection:
			cr := &request.ConnectionRequest{}
			cr.Init(reqTable.Bytes, reqTable.Pos)
			handleConnectionReq(con, cr)
		}
	}
}

func handleRegistrationReq(con net.Conn, r *request.RegistrationRequest) {
	name := string(r.Name())
	remoteAddr, err := helpers.StrToAddrV4(con.RemoteAddr().String())
	if err != nil {
		fmt.Printf("failed to parse remote address, err: %v\n", err)
		return
	}
	localAddr := helpers.ReqAddrToAddrV4(r.LocalAddr(&request.Addr{}))
	p := helpers.CreatePeer(name, remoteAddr, localAddr)
	fmt.Printf("adding new peer: name=%v local=%v remote=%v:%v\n", name, con.RemoteAddr(), localAddr.Addr, localAddr.Port)
	addPeer(name, p, con)

	_, err = con.Write(p.Table().Bytes)
	if err != nil {
		fmt.Printf("failed to send registration details, err: %v\n", err)
		return
	}
}

func handleConnectionReq(con net.Conn, r *request.ConnectionRequest) {
	requester := string(r.Requester())
	target := string(r.Peer())

	fmt.Printf("get connection request: from=%v to=%v\n", requester, target)

	targetPeer, tpConn, ok := getPeer(target)
	if !ok {
		fmt.Printf("target peer does not exist: peer=%v\n", target)
		con.Write([]byte{1}) // mark not found
		return
	}

	requesterPeer, _, ok := getPeer(requester)
	if !ok {
		fmt.Printf("requester is not registered: peer=%v\n", requester)
		con.Write([]byte{2}) // mark not registered yet
		return
	}

	fmt.Printf("sending details to target peer: peer=%v\n", target)
	_, err := tpConn.Write(requesterPeer.Table().Bytes)
	if err != nil {
		fmt.Printf("failed to send requester peer details to target peer, err: %v\n", err)
	}

	fmt.Printf("sending details to requester peer: peer=%v\n", requester)
	_, err = con.Write(targetPeer.Table().Bytes)
	if err != nil {
		fmt.Printf("failed to send target peer details to requester, err: %v\n", err)
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
