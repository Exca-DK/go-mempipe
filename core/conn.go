package conn

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Exca-DK/go-mempipe/core/links"
	"github.com/Exca-DK/go-mempipe/core/primitives"
)

var (
	errPlainMessageTooLarge = errors.New("message too large")
	errReadOnly             = errors.New("read only")
)

type Conn struct {
	cantWrite bool
	conn      *primitives.SharedMemMount
	mem       *primitives.SharedMem
	session   *sessionState
}

type sessionState struct {
	attached                    uint32
	wc, rc                      uint32
	rbuf                        readBuffer
	wbuf                        writeBuffer
	writeDeadline, readDeadline time.Duration
}

func NewReadOnlyConn(mnt *primitives.SharedMem) (*Conn, error) {
	var session sessionState
	session.readDeadline = -1
	session.writeDeadline = -1

	shm, err := mnt.Attach(&primitives.SHMAttachFlags{ReadOnly: false})
	if err != nil {
		return nil, err
	}

	c := &Conn{
		conn:    shm,
		session: &session,
		mem:     mnt,
	}

	c.session.attached = c.getRefreshAttachC()
	c.cantWrite = true

	return c, nil
}

func NewWriteOnlyConn(mnt *primitives.SharedMem) (*Conn, error) {
	var session sessionState
	session.readDeadline = -1
	session.writeDeadline = -1

	shm, err := mnt.Attach(&primitives.SHMAttachFlags{ReadOnly: false})
	if err != nil {
		return nil, err
	}

	c := &Conn{
		conn:    shm,
		session: &session,
		mem:     mnt,
	}
	c.session.attached = c.getRefreshAttachC()
	return c, nil
}

func NewConn(mnt *primitives.SharedMemMount) *Conn {
	var session sessionState
	session.readDeadline = -1
	session.writeDeadline = -1
	return &Conn{
		conn:    mnt,
		session: &session,
	}
}

// Read reads a message from the connection.
// The returned data buffer is valid until the next call to Read.
func (c *Conn) Read() (uint32, []byte, int, error) {
	if err := c.session.WaitRead(c.conn); err != nil {
		return 0, nil, 0, err
	}

	frame, err := c.session.readFrame(c.conn)
	if err != nil {
		return 0, nil, 0, err
	}

	code, data := frameIntoCodeAndData(frame)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("invalid message code: %v", err)
	}

	return uint32(code), data, len(data) + 4, err
}

func (h *sessionState) readFrame(conn *primitives.SharedMemMount) ([]byte, error) {
	h.rbuf.reset()
	conn.Seek(8, 0)

	size, err := conn.AtomicReadUint32()
	if err != nil {
		return nil, err
	}

	frame, err := h.rbuf.read(conn, int(size))
	if err != nil {
		return nil, err
	}

	h.rc++
	if h.rc == math.MaxUint32 {
		h.rc = 1
	}
	conn.Seek(4, 0)
	conn.AtomicWriteUint32(h.rc)
	return frame, nil
}

func (c *Conn) updateAttach(val uint32) {
	c.session.attached = val
}

func (c *Conn) getRefreshAttachC() uint32 {
	info, err := c.mem.Stat()
	if err != nil {
		panic(err)
	}
	return uint32(info.CurrentAttaches) - 1 //dont include self
}

func (c *Conn) Write(code uint32, data []byte) (uint32, error) {
	if len(data) > maxUint24 {
		return 0, errPlainMessageTooLarge
	}

	if c.cantWrite {
		return 0, errReadOnly
	}

	if err := c.session.WaitWrite(c.conn); err != nil {
		return 0, err
	}

	wireSize := uint32(len(data)) + 4
	return wireSize, c.session.writeFrame(c.conn, code, data)
}

var (
	ErrWriteTimedout = errors.New("write timedout")
	ErrReadTimedout  = errors.New("read timedout")
)

func (h *sessionState) SetWriteDeadline(deadline time.Duration) {
	h.writeDeadline = deadline
}

func (h *sessionState) SetReadDeadline(deadline time.Duration) {
	h.readDeadline = deadline
}

func (h *sessionState) writeFrame(conn *primitives.SharedMemMount, code uint32, data []byte) error {
	h.wbuf.reset()
	conn.Seek(8, 0)

	//signal datasize
	conn.AtomicWriteUint32(uint32(len(data) + 4))
	//write data
	appendUint32(h.wbuf.appendZero(4), int(code))
	h.wbuf.Write(data)
	_, err := conn.Write(h.wbuf.data)
	if err != nil {
		return err
	}

	//update the counter
	conn.Seek(0, 0)
	h.wc++
	if h.rc == math.MaxUint32 {
		h.wc = 1
	}
	return conn.AtomicWriteUint32(h.wc)
}

// Close closes the underlying network connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

func (h *sessionState) WaitWrite(conn *primitives.SharedMemMount) error {
	if (h.wc + h.rc) == 0 {
		return nil
	}

	ts := time.Now()
	i := 1
	for {
		i++
		if !h.canWrite(conn) {
			if i%10000 == 0 && h.writeDeadline != -1 {
				i = 1
				if time.Since(ts) > h.writeDeadline {
					return ErrWriteTimedout
				}
			}
		} else {
			break
		}

		links.Wait()
	}

	return nil
}

func (h *sessionState) WaitRead(conn *primitives.SharedMemMount) error {
	ts := time.Now()
	i := 1
	for {
		i++
		if !h.canRead(conn) {
			if i%1000 == 0 && h.readDeadline != -1 {
				i = 1
				if time.Since(ts) > h.readDeadline {
					return ErrReadTimedout
				}
			}
		} else {
			break
		}
		links.Wait()
	}
	return nil
}

func (h *sessionState) canWrite(conn *primitives.SharedMemMount) bool {
	conn.Seek(4, 0)
	c, err := conn.AtomicReadUint32()
	if err != nil {
		return false
	}
	if c == h.rc {
		return false
	}

	if h.rc+h.attached != c {
		return false
	}

	h.rc = c //update local read counter
	return true
}

func (h *sessionState) canRead(conn *primitives.SharedMemMount) bool {
	conn.Seek(0, 0)
	c, err := conn.AtomicReadUint32()
	if err != nil {
		return false
	}
	if c == h.wc {
		return false
	}
	h.wc = c //update local write counter
	return true
}
