package gotham

import (
	"bufio"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

var (
	addr                       = ":9001"
	dialCount                   = 5 // clients num

	waitTime = time.Millisecond * 5
)

func TestServe(t *testing.T) {
	ln, err := net.Listen("tcp", ":9001")
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{ReadTimeout: time.Minute}

	// starting the server
	go server.Serve(ln)
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup

	for i := 0; i < dialCount; i++ {
		wg.Add(1)
		go func(idx int) {
			conn, err := net.Dial("tcp", ":9001")
			if err != nil {
				t.Fatal(err)
			}

			defer conn.Close()
			defer wg.Done()

			ping := &PingMsg{
				Message: "Hello",
			}
			content, _ := proto.Marshal(ping)

			any := &any.Any{
				TypeUrl: proto.MessageName(ping),
				Value:   content,
			}

			payload, _ := proto.Marshal(any)

			w := bufio.NewWriter(conn)
			_ = WriteData(w, payload)
			_ = WriteData(w, payload)
			_ = w.Flush()

			time.Sleep(waitTime)
		}(i)
	}
	wg.Wait()
	time.Sleep(waitTime)
}
