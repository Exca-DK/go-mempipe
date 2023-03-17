package conn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Exca-DK/go-mempipe/core/primitives"
)

type tsStruct struct {
	Ts int64 `json:"ts"`
	Id int   `json:"id"`
}

func TestMemConn(t *testing.T) {
	conn1 := connSetup(t, true, 4096)
	conn2 := connSetup(t, false, 4096)
	defer conn2.Close()
	defer conn1.Close()

	const (
		iterations = 1024 * 5
		tcode      = 100
	)

	readTimings := make([]time.Duration, iterations)
	writeTimings := make([]time.Duration, iterations)

	var total uint64 = 0
	for i := 0; i < iterations; i++ {
		go func(i int) {
			var g uint64 = 0
			for x := 0; i < x; x++ {
				time.Sleep(10 * time.Microsecond)
				g += uint64(x)
			}
			_ = g
			atomic.AddUint64(&total, g)
		}(i)
	}

	var errCh chan error = make(chan error, 2)
	var doneCh chan struct{} = make(chan struct{}, 2)
	go func(conn *Conn) {
		var buff bytes.Buffer
		var obj tsStruct
		dec := json.NewDecoder(&buff)
		gts := time.Now()
		for i := 0; i < iterations; i++ {
			code, data, _, err := conn.Read()
			if err != nil {
				errCh <- fmt.Errorf("failed read conn. err: %v", err)
				return
			}

			if code != tcode {
				errCh <- fmt.Errorf("different code got. exp: %v, got: %v", tcode, code)
				return
			}

			_, err = buff.Write(data)
			if err != nil {
				errCh <- fmt.Errorf("Write failure. err: %v", err)
				return
			}

			if err := dec.Decode(&obj); err != nil {
				errCh <- fmt.Errorf("decode failure %v\n", err)
				return
			}
			duration := time.Unix(0, time.Now().UnixNano()).Sub(time.Unix(0, obj.Ts))
			readTimings[i] = duration
		}
		t.Logf("elapsed reading: %v\n", time.Since(gts))
		doneCh <- struct{}{}
	}(conn1)

	go func(conn *Conn) {

		var buffer bytes.Buffer
		enc := json.NewEncoder(&buffer)
		gts := time.Now()
		for i := 0; i < iterations; i++ {
			buffer.Reset()
			ts := time.Now()
			obj := tsStruct{Ts: ts.UnixNano(), Id: i}
			if err := enc.Encode(obj); err != nil {
				errCh <- fmt.Errorf("encoding failure %v\n", err)
				return
			}
			_, err := conn.Write(tcode, buffer.Bytes())
			if err != nil {
				errCh <- fmt.Errorf("failed Write conn. err: %v", err)
				return
			}
			duration := time.Unix(0, time.Now().UnixNano()).Sub(time.Unix(0, obj.Ts))
			writeTimings[i] = duration
		}
		t.Logf("elapsed reading: %v\n", time.Since(gts))
		doneCh <- struct{}{}
	}(conn2)

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			t.Fatalf("got error: %v", err)
		case <-doneCh:
		}
	}

	var avgR time.Duration = 0
	var avgW time.Duration = 0

	for i := 0; i < len(readTimings); i++ {
		readT, writeT := readTimings[i], writeTimings[i]
		fmt.Printf("read: %v, write: %v, i: %v\n", readT, writeT, i)
		avgR += readT
		avgW += writeT
	}

	t.Logf("avg write time: %v\n", avgR/time.Duration(iterations))
	t.Logf("avg read time: %v\n", avgW/time.Duration(iterations))
	t.Logf("iterations: %v\n", iterations)
	t.Logf("fake: %v\n", atomic.LoadUint64(&total))
}

type itest interface {
	Fatal(...any)
}

func BenchmarkMemConn(b *testing.B) {
	conn1 := connSetup(b, true, 4096)
	conn2 := connSetup(b, false, 4096)
	defer conn2.Close()
	defer conn1.Close()
	conn1.updateAttach(conn1.getRefreshAttachC())
	conn2.updateAttach(conn2.getRefreshAttachC())

	writeData := []byte("test interprocess message x1234567890")

	var d []byte

	for i := 0; i < b.N; i++ {
		_, err := conn2.Write(1, writeData)
		if err != nil {
			b.Fatal(b)
		}

		_, data, _, err := conn1.Read()
		if err != nil {
			b.Fatal(b)
		}
		if string(data) != string(writeData) {
			b.Fatalf("diff data: got %v, wanted: %v\n", string(data), string(writeData))
		}
		d = data
	}

	_ = d
}

func connSetup(t itest, create bool, size uint64) *Conn {
	mem, err := primitives.GetSharedMem(0xE4CAB, size, &primitives.SHMFlags{
		Create:    create,
		Exclusive: create,
		Perms:     0600,
	})
	if err != nil {
		t.Fatal(err)
	}

	mnt, err := mem.Attach(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !create {
		err = mem.Remove()
		if err != nil {
			t.Fatal(err)
		}
	}

	c := NewConn(mnt)
	c.mem = mem
	return c
}
