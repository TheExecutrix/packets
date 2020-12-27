package packets

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

func TestCreatePacketWritePacket(t *testing.T) {
	tests := []struct {
		input  []byte
		packet []byte
	}{
		{
			input:  nil,
			packet: []byte{0},
		},
		{
			input:  []byte{1, 2, 3, 4},
			packet: []byte{4, 1, 2, 3, 4},
		},
		{
			input: []byte("000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000000000"),
			packet: append([]byte{130, 2},
				[]byte("000000000000000000000000000000000000000000000000000000000000"+
					"000000000000000000000000000000000000000000000000000000000000000000"+
					"000000000000000000000000000000000000000000000000000000000000000000"+
					"000000000000000000000000000000000000000000000000000000000000000000")...),
		},
	}
	for _, test := range tests {
		packet := CreatePacket(test.input)
		if !bytes.Equal(packet, test.packet) {
			t.Errorf("Want: %+v and got: %+v for input %+v",
				test.packet, packet, test.input)
		}
		r, w := io.Pipe()
		go func() {
			if err := WritePacket(w, test.input); err != nil {
				t.Errorf("Error writing packet: %+v", test.input)
			}
			_ = w.Close()
		}()
		packet, err := ioutil.ReadAll(r)
		if err != nil {
			t.Errorf("Error reading: %v", err)
		}
		if !bytes.Equal(packet, test.packet) {
			t.Errorf("Want: %+v, got buffer: %+v for input %+v",
				test.packet, packet, test.input)
		}
	}
}

type errorWriter struct{}

func (ew *errorWriter) Write(data []byte) (int, error) {
	return 0, fmt.Errorf("test error")
}

func TestWritePacketError(t *testing.T) {
	ew := &errorWriter{}
	if err := WritePacket(ew, []byte("text")); err == nil {
		t.Errorf("No error in WritePacketError")
	} else if err.Error() != "test error" {
		t.Errorf("WritePacketError wanted \"test error\" got %v", err)
	}
}

func TestSplitPacket(t *testing.T) {
	tests := []struct {
		input    []byte
		packet   []byte
		overflow []byte
		err      error
	}{
		{
			input:    nil,
			packet:   nil,
			overflow: nil,
			err:      ErrTooShort,
		},
		{
			input:    []byte{1, 4},
			packet:   []byte{4},
			overflow: nil,
			err:      nil,
		},
		{
			input:    []byte{1, 5, 6},
			packet:   []byte{5},
			overflow: []byte{6},
			err:      nil,
		},
		{
			input:    []byte{10, 5, 6},
			packet:   nil,
			overflow: nil,
			err:      ErrTooShort,
		},
		{
			input:    []byte{0, 5, 6},
			overflow: []byte{5, 6},
		},
		{
			input: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 0},
			err: ErrOverflow,
		},
		{
			input: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255},
			err: ErrTooShort,
		},
		{
			input: []byte{255},
			err:   ErrTooShort,
		},
		{
			input: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 0},
			err:   ErrOverflow,
		},
		{
			input: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 1, 0},
			err:   ErrOverflow,
		},
	}
	for _, test := range tests {
		packet, overflow, err := SplitPacket(test.input)
		if err != test.err {
			t.Errorf("Wanted %v got %v for error in %+v", test.err, err, test.input)
		}
		if !bytes.Equal(packet, test.packet) {
			t.Errorf("Wanted %+v got %+v for packet in %+v", test.packet, packet, test.input)
		}
		if !bytes.Equal(overflow, test.overflow) {
			t.Errorf("Wanted %+v got %+v for overflow in %+v", test.overflow, overflow, test.input)
		}
	}
}

func TestPacketReader(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		if err := WritePacket(w, []byte{1}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		if err := WritePacket(w, []byte{2}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		_ = w.Close()
	}()
	packetReader := NewPacketReaderSize(r, 10)
	if len(packetReader.buffer) != 10 {
		t.Errorf("Error buffer invalid: 10 != %v", len(packetReader.buffer))
	}
	p1, err := packetReader.Read()
	if err != nil {
		t.Errorf("Packet error: %v", err)
	}
	if !bytes.Equal(p1, []byte{1}) {
		t.Errorf("Packet invalid got: %+v want: [1]", p1)
	}
	p2, err := packetReader.Read()
	if err != nil {
		t.Errorf("Packet error: %v", err)
	}
	if !bytes.Equal(p2, []byte{2}) {
		t.Errorf("Packet invalid got: %+v want: [2]", p2)
	}
	p3, err := packetReader.Read()
	if len(p3) != 0 {
		t.Errorf("Last packet non-empty")
	}
	if err != io.EOF {
		t.Errorf("End error invalid: %v", err)
	}
}

func TestPacketReaderError(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		if _, err := w.Write([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
			255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
			255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
			255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		_ = w.Close()
	}()
	packetReader := NewPacketReaderSize(r, 10)
	if len(packetReader.buffer) != 10 {
		t.Errorf("Error buffer invalid: got: %v want: 10", len(packetReader.buffer))
	}
	p, err := packetReader.Read()
	if err != ErrOverflow {
		t.Errorf("Error got: %v want: overflow", err)
	}
	if len(p) != 0 {
		t.Errorf("Error packet length invalid")
	}
}

func TestPacketReaderSize(t *testing.T) {
	r, w := io.Pipe()
	defer func() {
		_ = w.Close()
	}()
	defaultReader := NewPacketReader(r)
	if len(defaultReader.buffer) != 1024 {
		t.Errorf("Error default reader size, want: 1024, got: %v", len(defaultReader.buffer))
	}
	defer func() {
		err := recover()
		if fmt.Sprintf("%+v", err) != "size must be greater than 0" {
			t.Errorf("Wanted: %q got: %q", "size must be greater than 0", err)
		}
	}()
	_ = NewPacketReaderSize(r, -1)
}

func TestPacketStream(t *testing.T) {
	testPacketStream(t, NewPacketStream, 1024)
}

func TestPacketStreamShort(t *testing.T) {
	testPacketStream(t, func(r io.Reader) *PacketStream {
		return NewPacketStreamSize(r, 10)
	}, 10)
}

func TestPacketStreamLimit(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		if err := WritePacket(w, []byte{1}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		if err := WritePacket(w, []byte{2}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		if err := WritePacket(w, []byte{3, 4, 5, 6, 7, 8, 9}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		_ = w.Close()
	}()
	stream := NewPacketStream(r)
	stream.SetMaxPacketLength(5)
	packets := [][]byte{{1}, {2}}
	index := 0
	for packet := range stream.GetCh() {
		if index >= len(packets) {
			t.Errorf("Error index: %d", index)
			return
		}
		if !bytes.Equal(packets[index], packet) {
			t.Errorf("Error for packet %d, wanted: %+v, got: %+v",
				index, packets[index], packet)
		}
		index++
	}
	if err := stream.Err(); err != ErrTooLong {
		t.Errorf("Error for stream, wanted: %v, got: %v", ErrTooLong, err)
	}
}

func testPacketStream(t *testing.T, factory func(r io.Reader) *PacketStream, expectSize int) {
	r, w := io.Pipe()
	go func() {
		if err := WritePacket(w, []byte{1}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		if err := WritePacket(w, []byte{2}); err != nil {
			t.Errorf("Error writing packet: %+v", err)
		}
		_ = w.Close()
	}()
	stream := factory(r)
	if len(stream.r.buffer) != expectSize {
		t.Errorf("Error, stream buffer size, wanted: %d got %d",
			expectSize, len(stream.r.buffer))
	}
	packets := [][]byte{{1}, {2}}
	index := 0
	for packet := range stream.GetCh() {
		if index >= len(packets) {
			t.Errorf("Error index: %d", index)
			return
		}
		if !bytes.Equal(packets[index], packet) {
			t.Errorf("Error for packet %d, wanted: %+v, got: %+v",
				index, packets[index], packet)
		}
		index++
	}
	if err := stream.Err(); err != io.EOF {
		t.Errorf("Error for stream, wanted: %v, got: %v", io.EOF, err)
	}
}
