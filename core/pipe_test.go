package conn

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestPipe(t *testing.T) {
	conn1 := connSetup(t, true, 1024*10)
	conn2 := connSetup(t, false, 1024*10)
	defer conn2.Close()
	defer conn1.Close()

	var (
		iters   = 100
		msgData = make([]byte, 1024)
	)

	rand.Read(msgData)

	buffer := bytes.NewBuffer(make([]byte, 0))
	for i := 0; i < 10; i++ {
		buffer.Write(msgData)
	}

	pipeWriter := newMemPipe(conn1)
	pipeRecv := newMemPipe(conn2)

	for i := 0; i < iters; i++ {
		srcMsg := NewMessage(10, buffer.Bytes(), buffer.Len())

		if err := pipeWriter.WriteMsg(srcMsg); err != nil {
			t.Fatalf("write msg error: %v", err)
		}

		msg, err := pipeRecv.ReadMsg()
		if err != nil {
			t.Fatalf("read msg error: %v", err)
		}

		if msg.Code != srcMsg.Code {
			t.Fatalf("diff code. got: %v, want: %v", msg.Code, srcMsg.Code)
		}

		if string(srcMsg.Payload) != buffer.String() {
			t.Fatalf("diff msg. got: %v, want: %v\n", string(srcMsg.Payload), string(msgData))
		}
	}
}
