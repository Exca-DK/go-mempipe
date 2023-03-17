package conn

import (
	"encoding/binary"
)

func appendUint32(buff []byte, v int) {
	binary.BigEndian.PutUint32(buff, uint32(v))
}

func bytesToInt(b []byte) int {
	size := binary.BigEndian.Uint32(b[:4])
	return int(size)
}

func frameIntoCodeAndData(b []byte) (int, []byte) {
	size := bytesToInt(b)
	return int(size), b[4:]
}
