package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/sleep2death/gotham"
	"github.com/sleep2death/gotham/pb"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randstr(length int) string {
	return stringWithCharset(length, charset)
}

func main() {
	cfg := gotham.Default()

	go func() {
		err := gotham.Serve(cfg)
		if err != nil {
			log.Panic(err)
		}
	}()

	var conns []net.Conn

	for count := 0; count < 3; count++ {
		conn, err := net.Dial("tcp", "localhost:8202")
		conns = append(conns, conn)

		if err != nil {
			log.Panic(err)
		}

	}

	time.Sleep(time.Second * 1)

	for i, conn := range conns {
		for j := 0; j < 10; j++ {
			str := randstr(rand.Intn(16)) + " --- " + strconv.Itoa(i)
			msg := &pb.Talk{Str: str}
			// msg := []byte(str)
			data, _ := proto.Marshal(msg)
			conn.Write(gotham.WriteFrame(data))
		}
		// time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	}

	time.Sleep(time.Second * 1)
	fmt.Println(gotham.Count())
}
