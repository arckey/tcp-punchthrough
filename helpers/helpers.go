package helpers

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"github.com/arckey/tcp-punchthrough/types/peer"
	"github.com/arckey/tcp-punchthrough/types/request"
	fb "github.com/google/flatbuffers/go"
)

func PanicIfErr(msg string, err error) {
	if err == nil {
		return
	}

	panic(fmt.Errorf("%v, err: %v", msg, err))
}

func ConfigureSocket(sock int) error {
	for opt := range sockOpts {
		if err := syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, opt, 1); err != nil {
			return err
		}
	}
	return nil
}

func StrToAddrV4(addr string) (*syscall.SockaddrInet4, error) {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed address: %v", addr)
	}
	ip, port := parts[0], parts[1]
	portno, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v, err: %v", addr, err)
	}
	ipParts := strings.Split(ip, ".")
	if len(ipParts) != 4 {
		return nil, fmt.Errorf("malformed address: %v", addr)
	}

	p1, err := strconv.Atoi(ipParts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v, err: %v", addr, err)
	}
	p2, err := strconv.Atoi(ipParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v, err: %v", addr, err)
	}
	p3, err := strconv.Atoi(ipParts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v, err: %v", addr, err)
	}
	p4, err := strconv.Atoi(ipParts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %v, err: %v", addr, err)
	}

	return &syscall.SockaddrInet4{
		Port: portno,
		Addr: [4]byte{
			byte(p1),
			byte(p2),
			byte(p3),
			byte(p4),
		},
	}, nil
}

func CreateRegistrationReq(name string, addr *syscall.SockaddrInet4) []byte {
	b := fb.NewBuilder(256)

	// create address
	pName := b.CreateString(name)
	ip := b.CreateByteVector(addr.Addr[:])

	request.AddrStart(b)
	request.AddrAddPort(b, int32(addr.Port))
	request.AddrAddIp(b, ip)
	pAddr := request.AddrEnd(b)

	request.RegistrationRequestStart(b)
	request.RegistrationRequestAddName(b, pName)
	request.RegistrationRequestAddLocalAddr(b, pAddr)
	rr := request.RegistrationRequestEnd(b)

	request.RequestStart(b)
	request.RequestAddType(b, request.RequestTypeRegistration)
	request.RequestAddRequest(b, rr)
	r := request.RequestEnd(b)

	b.Finish(r)

	return b.Bytes[b.Head():]
}

func CreateConnectionRequest(target, requester string) []byte {
	b := fb.NewBuilder(0)
	t := b.CreateString(target)
	rq := b.CreateString(requester)
	request.ConnectionRequestStart(b)
	request.ConnectionRequestAddPeer(b, t)
	request.ConnectionRequestAddRequester(b, rq)
	cr := request.ConnectionRequestEnd(b)

	request.RequestStart(b)
	request.RequestAddType(b, request.RequestTypeConnection)
	request.RequestAddRequest(b, cr)
	r := request.RequestEnd(b)

	b.Finish(r)

	return b.Bytes[b.Head():]
}

func addAddr(b *fb.Builder, addr *syscall.SockaddrInet4) fb.UOffsetT {
	ip := b.CreateByteVector(addr.Addr[:])
	peer.AddrStart(b)
	peer.AddrAddIp(b, ip)
	peer.AddrAddPort(b, int32(addr.Port))
	return peer.AddrEnd(b)
}

func CreatePeer(name string, remoteAddr, localAddr *syscall.SockaddrInet4) *peer.Peer {
	b := fb.NewBuilder(256)
	n := b.CreateString(name)
	laddr := addAddr(b, localAddr)
	raddr := addAddr(b, remoteAddr)

	peer.PeerStart(b)
	peer.PeerAddName(b, n)
	peer.PeerAddLocalAddr(b, laddr)
	peer.PeerAddRemoteAddr(b, raddr)
	p := peer.PeerEnd(b)

	b.Finish(p)

	return peer.GetRootAsPeer(b.FinishedBytes(), p)
}

func PeerAddrToStr(addr *peer.Addr) string {
	return fmt.Sprintf("%v.%v.%v.%v:%v",
		addr.Ip(0),
		addr.Ip(1),
		addr.Ip(2),
		addr.Ip(3),
		addr.Port(),
	)
}

func PeerAddrToAddrV4(addr *peer.Addr) *syscall.SockaddrInet4 {
	return &syscall.SockaddrInet4{
		Addr: [4]byte{
			byte(addr.Ip(0)),
			byte(addr.Ip(1)),
			byte(addr.Ip(2)),
			byte(addr.Ip(3)),
		},
		Port: int(addr.Port()),
	}
}

func ReqAddrToAddrV4(addr *request.Addr) *syscall.SockaddrInet4 {
	return &syscall.SockaddrInet4{
		Addr: [4]byte{
			byte(addr.Ip(0)),
			byte(addr.Ip(1)),
			byte(addr.Ip(2)),
			byte(addr.Ip(3)),
		},
		Port: int(addr.Port()),
	}
}
