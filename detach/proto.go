//go:build unix

package detach

import (
	"encoding/binary"
	"fmt"
	"io"
)

// The socket carries frames: type (1 byte), length (2 bytes, little endian), payload.
const (
	frameData   = 'D' // both ways: terminal bytes (server -> client: output, client -> server: keys)
	frameResize = 'R' // client -> server: rows, cols (2 bytes each); also the first frame of an attach

	maxPayload = 1<<16 - 1
	chunk      = 16 << 10
)

func writeFrame(w io.Writer, typ byte, payload []byte) error {
	if len(payload) > maxPayload {
		return fmt.Errorf("detach: frame of %d bytes", len(payload))
	}
	b := make([]byte, 3+len(payload))
	b[0] = typ
	binary.LittleEndian.PutUint16(b[1:], uint16(len(payload)))
	copy(b[3:], payload)
	_, err := w.Write(b)
	return err
}

// writeData sends p as data frames.
func writeData(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n := min(len(p), chunk)
		if err := writeFrame(w, frameData, p[:n]); err != nil {
			return err
		}
		p = p[n:]
	}
	return nil
}

func readFrame(r io.Reader) (typ byte, payload []byte, err error) {
	var h [3]byte
	if _, err = io.ReadFull(r, h[:]); err != nil {
		return 0, nil, err
	}
	payload = make([]byte, binary.LittleEndian.Uint16(h[1:]))
	if _, err = io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return h[0], payload, nil
}

func resizePayload(rows, cols uint16) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint16(b, rows)
	binary.LittleEndian.PutUint16(b[2:], cols)
	return b
}
