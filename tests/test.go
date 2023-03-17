package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	core "github.com/Exca-DK/go-mempipe/core"
)

var __id int64 = 0xE4CA

var (
	serverOk   = flag.Bool("server", false, "")
	iterations = 100
	id         = flag.Int("id", 1, "")
)

type tsStruct struct {
	Ts   int64  `json:"ts"`
	Data []byte `json:"data"`
	Id   int    `json:"id"`
}

func startServer() {
	pipe, err := core.NewMemWritePipe(__id, 1024*2)
	if err != nil {
		core.ClearPipe(__id)
		log.Fatalf("server failure start: %v", err)
	}
	defer pipe.Close()
	pipe.SetWriteDeadline(10 * time.Second)
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)

	var (
		msgData = make([]byte, 1024)
	)
	rand.Read(msgData)
	writeTimings := make([]time.Duration, iterations)

	fmt.Printf("waiting for conn\n")
	pipe.WaitConn()
	fmt.Printf("sb connected\n")

	for i := 0; i < iterations; i++ {
		ts := time.Now()
		obj := tsStruct{Ts: ts.UnixNano(), Id: i, Data: msgData}
		if err := enc.Encode(obj); err != nil {
			fmt.Printf("encoding failure %v\n", err)
			return
		}

		err := pipe.WriteMsg(core.NewMessage(10, buffer.Bytes(), buffer.Len()))
		if err != nil {
			fmt.Printf("failed Write conn. err: %v", err)
			return
		}

		duration := time.Unix(0, time.Now().UnixNano()).Sub(time.Unix(0, obj.Ts))
		writeTimings[i] = duration
		buffer.Reset()
	}
	avgDuration := time.Duration(0)
	for _, duration := range writeTimings {
		// fmt.Printf("i: %v duration: %v\n", i, duration)
		avgDuration += duration
	}

	fmt.Printf("avg: %v\n", avgDuration/time.Duration(len(writeTimings)))
}

func startClient(id int) {
	pipe, err := core.NewMemReadPipe(__id, 1024*2)
	if err != nil {
		log.Fatalf("client failure start: %v", err)
	}
	defer pipe.Close()

	readTimings := make([]time.Duration, iterations)
	var buff bytes.Buffer
	var obj tsStruct
	dec := json.NewDecoder(&buff)
	for i := 0; i < iterations; i++ {
		msg, err := pipe.ReadMsg()
		ts := time.Now()
		if err != nil {
			fmt.Printf("failed read conn. err: %v", err)
			return
		}
		if _, err := buff.Write(msg.Payload); err != nil {
			fmt.Printf("Write failure %v\n", err)
			return
		}

		if err := dec.Decode(&obj); err != nil {
			fmt.Printf("decode failure %v\n", err)
			return
		}

		duration := time.Unix(0, ts.UnixNano()).Sub(time.Unix(0, obj.Ts))
		readTimings[i] = duration
		buff.Reset()
	}
	avgDuration := time.Duration(0)
	for i, duration := range readTimings {
		fmt.Printf("i: %v duration: %v\n", i, duration)
		avgDuration += duration
	}

	fmt.Printf("avg: %v\n", avgDuration/time.Duration(len(readTimings)))
}

func main() {
	flag.Parse()
	if serverOk != nil && *serverOk {
		startServer()
	} else {
		startClient(*id)
	}
}
