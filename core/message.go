package conn

import (
	"fmt"
	"time"
)

type Decoder interface {
	Decode([]byte) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type Msg struct {
	Code       uint64
	Size       uint32 // Size of the raw payload
	Payload    []byte
	ReceivedAt int64
}

func (msg Msg) Time() time.Time {
	return time.UnixMicro(msg.ReceivedAt)
}

func (msg *Msg) setTimestamp(t time.Time) {
	msg.ReceivedAt = t.UnixMicro()
}

func NewMessage(code uint64, payload []byte, size int) Msg {
	return Msg{
		Code:       code,
		Size:       uint32(size),
		Payload:    payload,
		ReceivedAt: 0,
	}
}

func (msg Msg) Decode(decoder Decoder) error {
	err := decoder.Decode(msg.Payload)
	if err != nil {
		return err
	}
	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("msg #%v (%v bytes)", msg.Code, msg.Size)
}

func (msg Msg) Discard() error {
	return nil
}

type MsgReader interface {
	ReadMsg() (Msg, error)

	// if fails to recv message within this window, ReadMsg will return error
	SetReadDeadline(t time.Duration)
}

type MsgWriter interface {
	WriteMsg(Msg) error

	// if fails to send message within this window, WriteMsg will return error
	SetWriteDeadline(t time.Duration)
}
