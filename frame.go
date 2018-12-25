package gotham

import "encoding/binary"

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
func WriteFrame(msg []byte) (data []byte) {
	sizeBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBuf, uint64(len(msg)))

	msg = append(sizeBuf, msg...)
	data = append(data, msg...)
	return
}
