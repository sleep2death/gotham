package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/sleep2death/gotham"
	"github.com/sleep2death/gotham/pb"
	"github.com/teris-io/shortid"
)

func main() {
	addr := ":4000"

	ln, err := net.Listen("tcp", addr)

	if err != nil {
		panic(err)
	}

	server := &gotham.Server{}
	server.ReadTimeout = time.Minute

	listen := func(ln net.Listener) {
		if err := server.Serve(ln); err != nil {
			// it is going to throw an error, when the server finally closed
			fmt.Println(ln.Addr(), err.Error())
		}
	}

	server.ServeTCP = func(w io.Writer, fh gotham.FrameHeader, fb []byte) {
		out := &pb.AddressBook{}
		if err := proto.Unmarshal(fb, out); err != nil {
			log.Fatalln("Failed to parse address book:", err)
		}

		for j := 0; j < len(out.People); j++ {
			people := out.People[j]
			fmt.Println(people)
		}
	}

	// serve two listeners
	go listen(ln)

	conn, err := net.DialTimeout("tcp", addr, time.Minute*5)

	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(conn)
	book := &pb.AddressBook{}

	var people []*pb.Person

	for i := 0; i < 10; i++ {
		id, _ := shortid.Generate()
		person := &pb.Person{
			Id:    id,
			Name:  "John Doe",
			Email: "jdoe@example.com",
			Phones: []*pb.Person_PhoneNumber{
				{Number: "555-4321", Type: pb.Person_HOME},
			},
			LastUpdated: ptypes.TimestampNow(),
		}
		people = append(people, person)
	}
	book.People = people
	in, err := proto.Marshal(book)

	if err != nil {
		log.Fatalln("Failed to encode address book:", err)
	}

	_ = gotham.WriteData(w, in)
	w.Flush()

	time.Sleep(time.Second)

}
