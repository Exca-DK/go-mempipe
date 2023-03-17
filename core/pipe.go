package conn

import (
	"fmt"
	"sync"
	"time"

	"github.com/Exca-DK/go-mempipe/core/primitives"
)

type Pipe interface {
	MsgReadWriter
	Close()    // closes mem attach
	WaitConn() // waits for client to attach
}

type pipe struct {
	rmu, wmu sync.Mutex
	conn     *Conn
}

//trigger this if failed to close the mem due to panic or other error
func ClearPipe(id int64) {
	prim, err := primitives.GetSharedMem(id, 0, &primitives.SHMFlags{Create: false, Exclusive: false, Perms: 0600})
	if err != nil {
		return
	}

	prim.Remove()
}

func (p *pipe) SetWriteDeadline(t time.Duration) {
	p.conn.session.SetWriteDeadline(t)
}

func (p *pipe) SetReadDeadline(t time.Duration) {
	p.conn.session.SetWriteDeadline(t)
}

// Is able to send messages. New message is only being sent when previous has been acknowledged by recv.
func NewMemWritePipe(id int64, size uint64) (Pipe, error) {
	prim, err := primitives.GetSharedMem(id, size, &primitives.SHMFlags{Create: true, Exclusive: true, Perms: 0600})
	if err != nil {
		return nil, err
	}

	conn, err := NewWriteOnlyConn(prim)
	if err != nil {
		return nil, err
	}
	return newMemPipe(conn), nil
}

// Can only recv messages.
func NewMemReadPipe(id int64, size uint64) (Pipe, error) {
	prim, err := primitives.GetSharedMem(id, size, &primitives.SHMFlags{Create: false, Exclusive: false, Perms: 0600})
	if err != nil {
		fmt.Println("123")
		return nil, err
	}

	conn, err := NewReadOnlyConn(prim)
	if err != nil {
		fmt.Println("321")
		return nil, err
	}
	prim.Remove()
	return newMemPipe(conn), nil
}

func newMemPipe(prim *Conn) *pipe {
	return &pipe{
		conn: prim,
	}
}

func (t *pipe) ReadMsg() (Msg, error) {
	t.rmu.Lock()
	defer t.rmu.Unlock()

	var msg Msg

	code, data, _, err := t.conn.Read()
	if err == nil {
		msg = Msg{
			Code:    uint64(code),
			Size:    uint32(len(data)),
			Payload: data,
		}
		msg.setTimestamp(time.Now())
	}
	return msg, err
}

func (t *pipe) WaitConn() {
	now := t.conn.session.attached
	if now != 0 {
		return
	}

	for {
		refreshed := t.conn.getRefreshAttachC()
		if refreshed != now {
			t.conn.updateAttach(refreshed)
			return
		}
		time.Sleep(1 * time.Second)
	}

}

func (t *pipe) WriteMsg(msg Msg) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()

	_, err := t.conn.Write(uint32(msg.Code), msg.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (t *pipe) Close() {
	t.wmu.Lock()
	defer t.wmu.Unlock()

	if t.conn != nil {
		t.conn.Close()
	}
}
