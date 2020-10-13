package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	. "github.com/arckey/tcp-punchthrough/helpers"
	"github.com/arckey/tcp-punchthrough/types/peer"
)

var sAddrFlag = flag.String("negotiator-addr", "", "the address of the negotiator server")
var peerNameFlag = flag.String("name", "", "the peer name, other peers will use it to connect to you")
var targetNameFlag = flag.String("target", "", "the name of the target peer you want to connect to")

const (
	connectRetries       = 3
	connectRetryDelay    = 2000 * time.Millisecond
	establishConnTimeout = 300 * time.Second
)

var localPort int

func main() {
	validateFlags()

	sock := connectToNegotiatorServer()

	if *targetNameFlag == "" {
		acceptIncommingPeer(sock)
	} else {
		p := requestPeer(sock, *targetNameFlag)
		s := establishConnectionToPeer(p)
		if s == -1 {
			panic(fmt.Errorf("failed to establish connection to peer"))
		}
		chatWithPeer(s, p)
	}
}

func chatWithPeer(sock int, p *peer.Peer) {
	buf := make([]byte, 256)
	pname := string(p.Name())
	fmt.Printf("connected to: %v, sock=%v\n", pname, sock)
	for {
		fmt.Printf("[msg:] ")
		n, err := os.Stdin.Read(buf)
		PanicIfErr("failed to read message from stdin", err)

		n, err = syscall.Write(sock, buf[:n])
		PanicIfErr("failed to write message", err)

		n, err = syscall.Read(sock, buf)
		PanicIfErr("failed to read response from peer", err)

		fmt.Printf("[resp:] %v", string(buf[0:n]))
	}
}

func establishConnectionToPeer(p *peer.Peer) int {
	fmt.Printf("trying to establish connection to: %v\n", string(p.Name()))
	acceptChan := make(chan int)
	connectLocalChan := make(chan int)
	connectRemoteChan := make(chan int)

	pname := string(p.Name())
	remoteAddr := PeerAddrToAddrV4(p.RemoteAddr(&peer.Addr{}))
	// localAddr := PeerAddrToAddrV4(p.LocalAddr(&peer.Addr{}))

	go attemptAccept(acceptChan)
	time.Sleep(time.Second * 1) // wait one second
	// go attemptConnect(localAddr, connectLocalChan)
	go attemptConnect(remoteAddr, connectRemoteChan)

	failures := 0
	tout := time.After(establishConnTimeout)
	for failures != 3 {
		select {
		case sock := <-acceptChan:
			return sock
		case sock := <-connectLocalChan:
			return sock
		case sock := <-connectRemoteChan:
			return sock
		case <-tout:
			fmt.Println("timeout reached")
			return -1
		}
	}

	fmt.Printf("all attempts to connect to: %v have failed\n", pname)
	return -1
}

func makeSock(localPort int) int {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_IP)
	PanicIfErr("failed to create socket", err)

	err = ConfigureSocket(sock)
	PanicIfErr("failed to configure socket", err)

	err = syscall.Bind(sock, &syscall.SockaddrInet4{Port: localPort})
	PanicIfErr("failed to bind socket", err)

	return sock
}

func attemptConnect(addr *syscall.SockaddrInet4, res chan int) {
	conChan := make(chan int)
	socks := make([]int, connectRetries)

	for i := range socks {
		sock := makeSock(localPort)
		fmt.Printf("creating socket: sock=%v\n", sock)
		socks[i] = sock
	}

	tryAgain := func(addr *syscall.SockaddrInet4, res chan int, try, sock int) {
		fmt.Printf("attempting to connect to %v:%v retry=%v sock=%v\n", addr.Addr, addr.Port, try, sock)
		if err := syscall.Connect(sock, addr); err != nil {
			fmt.Printf("failed to connect to %v:%v, retry=%v, err=%v\n", addr.Addr, addr.Port, try, err)
		} else {
			fmt.Printf("succefully connected to %v:%v\n", addr.Addr, addr.Port)
			res <- sock
			return
		}
	}

	go tryAgain(addr, conChan, 0, socks[0])

	for i := 1; i < connectRetries; i++ {
		tagain := time.After(connectRetryDelay)
		select {
		case successSock := <-conChan:
			res <- successSock
			return
		case <-tagain:
			go tryAgain(addr, conChan, i, socks[i])
		}
	}
}

func attemptAccept(res chan int) {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_IP)
	PanicIfErr("failed to create socket", err)

	err = ConfigureSocket(sock)
	PanicIfErr("failed to configure socket", err)

	sa := &syscall.SockaddrInet4{Port: localPort}
	copy(sa.Addr[:], net.ParseIP("0.0.0.0"))
	err = syscall.Bind(sock, sa)
	PanicIfErr("failed to bind socket", err)

	err = syscall.Listen(sock, 10)
	PanicIfErr("failed to listen with socket", err)

	fmt.Printf("listening for incomming connections on port %v, sock=%v\n", localPort, sock)
	peerSock, peerAddr, err := syscall.Accept(sock)
	if err != nil {
		fmt.Printf("failed to accept connection, err: %v\n", err)
		return
	}
	peerAddrV4, _ := peerAddr.(*syscall.SockaddrInet4)
	fmt.Printf("accepted connection from: %v:%v\n", peerAddrV4.Addr, peerAddrV4.Port)
	res <- peerSock
}

func requestPeer(sock int, targetPeer string) *peer.Peer {
	buf := make([]byte, 512)
	cr := CreateConnectionRequest(targetPeer, *peerNameFlag)
	_, err := syscall.Write(sock, cr)
	PanicIfErr("failed to send connection request", err)

	n, err := syscall.Read(sock, buf)
	PanicIfErr("failed to read peer from server", err)
	if n == 1 && buf[0] == 1 {
		panic(fmt.Errorf("peer with name %v was not found", targetPeer))
	}

	return peer.GetRootAsPeer(buf[:n], 0)
}

func acceptIncommingPeer(sock int) {
	buf := make([]byte, 512)
	fmt.Println("waiting for incomming peer requests")
	for {
		n, err := syscall.Read(sock, buf)
		PanicIfErr("failed to read from negotiator server", err)

		other := peer.GetRootAsPeer(buf[:n], 0)
		name := string(other.Name())
		remoteAddr := other.RemoteAddr(&peer.Addr{})
		localAddr := other.LocalAddr(&peer.Addr{})
		fmt.Printf("got connection request from: name=%v local=%v remote=%v\n",
			name,
			PeerAddrToStr(localAddr),
			PeerAddrToStr(remoteAddr))
		peerSock := establishConnectionToPeer(other)
		if peerSock == -1 {
			panic(fmt.Errorf("failed to establish connection to peer"))
		}
		go handlePeerConnection(peerSock, other)
	}
}

func handlePeerConnection(peerSock int, p *peer.Peer) {
	buf := make([]byte, 512)
	pname := string(p.Name())

	// echo server
	for {
		n, err := syscall.Read(peerSock, buf)
		if err != nil {
			fmt.Printf("failed to read from peer: %v, err: %v\n", pname, err)
			return
		}

		fmt.Printf("[%v:] %v\n", pname, string(buf[:n]))
		_, err = syscall.Write(peerSock, buf[:n])
		if err != nil {
			fmt.Printf("failed to respond to peer: %v, err: %v\n", pname, err)
			return
		}
	}
}

func connectToNegotiatorServer() int {
	buf := make([]byte, 512)
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_IP)
	PanicIfErr("failed to create socket", err)

	err = ConfigureSocket(sock)
	PanicIfErr("failed to configure socket", err)

	err = syscall.Bind(sock, &syscall.SockaddrInet4{})
	PanicIfErr("failed to bind socket", err)

	laddr, err := syscall.Getsockname(sock)
	PanicIfErr("failed to get local address", err)
	laddrv4 := laddr.(*syscall.SockaddrInet4)
	localPort = laddrv4.Port

	sAddr, err := StrToAddrV4(*sAddrFlag)
	PanicIfErr("failed to parse negotiator address", err)

	err = syscall.Connect(sock, sAddr)
	PanicIfErr("failed to connect to negotiator server", err)
	fmt.Printf("connected to negotiator server using local address: %v:%v\n", laddrv4.Addr, laddrv4.Port)

	req := CreateRegistrationReq(*peerNameFlag, laddrv4)
	_, err = syscall.Write(sock, req)
	PanicIfErr("failed to register to negotiator", err)
	fmt.Printf("registered as: %v, %v:%v\n", *peerNameFlag, laddrv4.Addr, laddrv4.Port)

	n, err := syscall.Read(sock, buf)
	PanicIfErr("failed to read from negotiator server", err)

	me := peer.GetRootAsPeer(buf[:n], 0)
	remoteAddr := me.RemoteAddr(&peer.Addr{})
	fmt.Printf("recognized as: %v\n", PeerAddrToStr(remoteAddr))

	return sock
}

func validateFlags() {
	flag.Parse()
	if *sAddrFlag == "" {
		panic("--negotiator-addr flag is required")
	}

	if *peerNameFlag == "" {
		panic("--name flag is required")
	}
}
