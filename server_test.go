package gotham

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

func TestServe(t *testing.T) {
	// dialCount := 5 // clients num
	waitTime := time.Millisecond * 5

	ln, err := net.Listen("tcp", ":9002")
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{ReadTimeout: time.Minute}

	// starting the server
	go server.Serve(ln)
	time.Sleep(time.Millisecond)

	conn, err := net.Dial("tcp", ":9002")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ping := &Ping{
		Message: "Hello",
	}
	content, _ := proto.Marshal(ping)

	any := &types.Any{
		TypeUrl: proto.MessageName(ping),
		Value:   content,
	}

	payload, _ := proto.Marshal(any)

	// write two payloads at once
	w := bufio.NewWriter(conn)
	WriteFrame(w, payload)
	WriteFrame(w, payload)
	w.Flush()

	// make the payload manually
	var flags Flags
	flags |= FlagFrameAck
	length := len(payload)

	header := [frameHeaderLen]byte{
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
		byte(FrameData),
		byte(flags),
	}

	// test incomplete header
	// write the broken header first
	w.Write(header[:3])
	w.Flush()
	// then wait a little while, write the left...
	time.Sleep(waitTime)
	wbuf := append(header[3:], payload...)
	w.Write(wbuf)
	w.Flush()

	time.Sleep(waitTime)

	// test incomplete body
	// write the broken body first
	wbuf = append(header[:frameHeaderLen], payload[:3]...)
	w.Write(wbuf)
	w.Flush()
	// then wait a little while, write the left...
	time.Sleep(waitTime)
	wbuf = append(payload[3:])
	w.Write(wbuf)
	w.Flush()

	// wait for server writing back
	time.Sleep(waitTime)
}
