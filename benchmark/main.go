package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sleep2death/gotham"
)

var (
	targetAddr  = flag.String("a", "127.0.0.1:8202", "target echo server address")
	testMsgLen  = flag.Int("l", 26, "test message length")
	testConnNum = flag.Int("c", 2000, "test connection number")
	testSeconds = flag.Int("t", 30, "test duration in seconds")
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

func readFrames(data []byte) (msgs [][]byte, leftover []byte) {
	leftover = data

	var batchSize int

	for {
		leftLen := len(leftover)
		// not enough header data
		if leftLen < 8 {
			break
		}

		batchSize = int(binary.BigEndian.Uint64(leftover[:8]))
		// not enough body data
		if batchSize >= leftLen+8 {
			break
		}

		leftover, msgs = leftover[8+batchSize:], append(msgs, leftover[8:batchSize+8])
	}

	return
}

// WriteFrame prefix the data with size header
func writeFrame(msg []byte) (data []byte) {
	sizeBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBuf, uint64(len(msg)))

	msg = append(sizeBuf, msg...)
	data = append(data, msg...)
	return
}

func main() {
	flag.Parse()

	var (
		outNum uint64
		inNum  uint64
		stop   uint64
	)

	cfg := gotham.Default()

	go func() {
		err := gotham.Serve(cfg)
		if err != nil {
			log.Panic(err)
		}
	}()

	go func() {
		time.Sleep(time.Second * time.Duration(*testSeconds))
		atomic.StoreUint64(&stop, 1)
	}()

	wg := new(sync.WaitGroup)

	for i := 0; i < *testConnNum; i++ {
		wg.Add(1)

		go func() {
			if conn, err := net.DialTimeout("tcp", "localhost:8202", time.Minute*99999); err == nil {
				for {
					// Write data
					str := "Echo Server"
					msg := []byte(str)
					conn.Write(writeFrame(msg))
					atomic.AddUint64(&outNum, 1)

					if atomic.LoadUint64(&stop) == 1 {
						break
					}

					// Read data
					bs := make([]byte, 1024)
					for {
						_, err = conn.Read(bs)

						if err != nil {
							break
						}

						msgs, leftover := readFrames(bs)

						atomic.AddUint64(&inNum, uint64(len(msgs)))

						l := len(leftover)

						if l > 0 {
							if l != len(bs) {
								bs = append(bs[:0], leftover...)
							}
						} else if l > 0 {
							bs = bs[:0]
						}
					}
				}
			} else {
				log.Println(err)
			}

			wg.Done()
		}()
	}

	wg.Wait()

	fmt.Println("Benchmarking:", *targetAddr)
	fmt.Println(*testConnNum, "clients, running", *testMsgLen, "bytes,", *testSeconds, "sec.")
	fmt.Println()
	fmt.Println("Speed:", outNum/uint64(*testSeconds), "request/sec,", inNum/uint64(*testSeconds), "response/sec")
	fmt.Println("Requests:", outNum)
	fmt.Println("Responses:", inNum)
}
