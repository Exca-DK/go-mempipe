package conn

import (
	"github.com/Exca-DK/go-mempipe/core/primitives"
)

type readBuffer struct {
	data []byte
	end  int
}

func (b *readBuffer) reset() {}

func (b *readBuffer) read(r *primitives.SharedMemMount, n int) ([]byte, error) {
	// Make buffer space available.
	b.grow(n)

	_, err := r.Read(b.data[0:n])
	if err != nil {
		return nil, err
	}

	return b.data[0:n], nil
}

func (b *readBuffer) grow(n int) {
	need := n - b.end
	if need <= 0 {
		return
	}

	b.data = append(b.data, make([]byte, need)...)
	b.end += need
}

type writeBuffer struct {
	data []byte
}

func (b *writeBuffer) reset() {
	b.data = b.data[:0]
}

func (b *writeBuffer) appendZero(n int) []byte {
	offset := len(b.data)
	b.data = append(b.data, make([]byte, n)...)
	return b.data[offset : offset+n]
}

func (b *writeBuffer) Write(data []byte) (int, error) {
	b.data = append(b.data, data...)
	return len(data), nil
}

const maxUint24 = int(^uint32(0) >> 8)
