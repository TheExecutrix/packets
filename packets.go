package packets

import (
	"encoding/binary"
	"github.com/theexecutrix/consterr"
	"io"
	"sync"
)

const (
	// ErrOverflow is returned if the packet is too long for the library.
	ErrOverflow = consterr.Error("packet overflow")
	// ErrTooShort is returned if the packet is too short.
	ErrTooShort = consterr.Error("data too short")
	// ErrTooLong is returned if the packet is too long.
	ErrTooLong = consterr.Error("data too long")
)

// MaxPacketHeaderLength is the maximum supported packet header size.
const MaxPacketHeaderLength = 16

// CreatePacket converts data to a packet.
func CreatePacket(data []byte) []byte {
	temp := make([]byte, MaxPacketHeaderLength)
	length := binary.PutUvarint(temp, uint64(len(data)))
	return append(temp[0:length], data...)
}

// WritePacket writes data as a packet to an io.Writer.
func WritePacket(w io.Writer, data []byte) error {
	temp := make([]byte, MaxPacketHeaderLength)
	length := binary.PutUvarint(temp, uint64(len(data)))
	if _, err := w.Write(temp[0:length]); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// SplitPacket splits packet data into the current packet and the remainder.
func SplitPacket(data []byte) ([]byte, []byte, error) {
	if len(data) == 0 {
		return nil, nil, ErrTooShort
	}
	if data[0] == 0 {
		return nil, data[1:], nil
	}
	length, headerLength := binary.Uvarint(data)
	if headerLength == 0 {
		return nil, nil, ErrTooShort
	}
	if headerLength < 0 {
		return nil, nil, ErrOverflow
	}
	if int(length) <= 0 {
		return nil, nil, ErrOverflow
	}
	if (int(length) + headerLength) <= 0 {
		return nil, nil, ErrOverflow
	}
	if (int(length) + headerLength) > len(data) {
		return nil, nil, ErrTooShort
	}
	return data[headerLength : headerLength+int(length)], data[headerLength+int(length):], nil
}

// PacketReader reads packets from an io.Reader.
type PacketReader struct {
	r               io.Reader
	data            []byte
	buffer          []byte
	maxPacketLength uint
}

// NewPacketReader creates PacketReader with 1024 byte buffer.
func NewPacketReader(r io.Reader) *PacketReader {
	return NewPacketReaderSize(r, 1024)
}

// NewPacketReaderSize creates PacketReader with set buffer size.
func NewPacketReaderSize(r io.Reader, size int) *PacketReader {
	if size <= 0 {
		panic("size must be greater than 0")
	}
	return &PacketReader{
		r:      r,
		buffer: make([]byte, size),
	}
}

// SetMaxPacketLength sets the maximum packet length, zero is unlimited.
func (r *PacketReader) SetMaxPacketLength(maxPacketLength uint) {
	r.maxPacketLength = maxPacketLength
}

// Read reads from the PacketReader.
func (r *PacketReader) Read() ([]byte, error) {
	for {
		if r.maxPacketLength > 0 && uint(len(r.data)) > r.maxPacketLength {
			return nil, ErrTooLong
		}
		data, next, err := SplitPacket(r.data)
		if err == nil {
			r.data = next
			return data, nil
		}
		if err != ErrTooShort {
			return nil, err
		}
		n, err := r.r.Read(r.buffer)
		if err != nil {
			return nil, err
		}
		r.data = append(r.data, r.buffer[0:n]...)
	}
}

// PacketStream streams packets from a reader or an error.
type PacketStream struct {
	mu  sync.Mutex
	r   *PacketReader
	ch  chan []byte
	err error
}

// NewPacketStream creates a PacketStream.
func NewPacketStream(r io.Reader) *PacketStream {
	stream := &PacketStream{
		ch: make(chan []byte),
		r:  NewPacketReader(r),
	}
	go stream.thread()
	return stream
}

// NewPacketStreamSize creates a PacketStream with set buffer size.
func NewPacketStreamSize(r io.Reader, size int) *PacketStream {
	stream := &PacketStream{
		ch: make(chan []byte),
		r:  NewPacketReaderSize(r, size),
	}
	go stream.thread()
	return stream
}

// GetCh gets the channel of packets from the stream. The channel is closed on error.
func (stream *PacketStream) GetCh() <-chan []byte {
	return stream.ch
}

// Err gets error that closed the PacketStream.
func (stream *PacketStream) Err() error {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	return stream.err
}

// SetMaxPacketLength sets the maximum packet length, zero is unlimited.
func (stream *PacketStream) SetMaxPacketLength(maxPacketLength uint) {
	stream.r.SetMaxPacketLength(maxPacketLength)
}

// thread is the thread for PacketStream.
func (stream *PacketStream) thread() {
	for {
		data, err := stream.r.Read()
		if err != nil {
			stream.mu.Lock()
			defer stream.mu.Unlock()
			stream.err = err
			close(stream.ch)
			return
		}
		stream.ch <- data
	}
}
