package bitio

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/icza/mighty"
)

func TestCountReader(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewCountReader(bytes.NewBuffer(data))
	eq, expEq := mighty.EqExpEq(t)

	eq(uint64(0), r.GetBitPosition())

	expEq(byte(3))(r.ReadByte())
	eq(uint64(8), r.GetBitPosition())

	expEq(uint64(255))(r.ReadBits(8))
	eq(uint64(16), r.GetBitPosition())

	expEq(uint64(0xc))(r.ReadBits(4))
	eq(uint64(20), r.GetBitPosition())

	expEq(uint64(0xc1))(r.ReadBits(8))
	eq(uint64(28), r.GetBitPosition())

	expEq(uint64(0xabcde))(r.ReadBits(20))
	eq(uint64(48), r.GetBitPosition())

	expEq(true)(r.ReadBool())
	eq(uint64(49), r.GetBitPosition())

	expEq(false)(r.ReadBool())
	eq(uint64(50), r.GetBitPosition())

	eq(uint8(6), r.Align())
	eq(uint64(56), r.GetBitPosition())

	s := make([]byte, 2)
	expEq(2)(r.Read(s))
	eq(uint64(72), r.GetBitPosition())

	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))

	expEq(uint64(0xf))(r.ReadBits(4))
	eq(uint64(76), r.GetBitPosition())

	expEq(2)(r.Read(s))
	eq(uint64(92), r.GetBitPosition())
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))
}

func TestCountReaderTry(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewCountReader(bytes.NewBuffer(data))
	eq := mighty.Eq(t)

	eq(uint64(0), r.GetBitPosition())

	eq(byte(3), r.TryReadByte())
	eq(uint64(8), r.GetBitPosition())

	eq(uint64(255), r.TryReadBits(8))
	eq(uint64(16), r.GetBitPosition())

	eq(uint64(0xc), r.TryReadBits(4))
	eq(uint64(20), r.GetBitPosition())

	eq(uint64(0xc1), r.TryReadBits(8))
	eq(uint64(28), r.GetBitPosition())

	eq(uint64(0xabcde), r.TryReadBits(20))
	eq(uint64(48), r.GetBitPosition())

	eq(true, r.TryReadBool())
	eq(false, r.TryReadBool())
	eq(uint64(50), r.GetBitPosition())

	eq(uint8(6), r.Align())
	eq(uint64(56), r.GetBitPosition())

	s := make([]byte, 2)
	eq(2, r.TryRead(s))
	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))
	eq(uint64(72), r.GetBitPosition())

	eq(uint64(0xf), r.TryReadBits(4))
	eq(uint64(76), r.GetBitPosition())

	eq(2, r.TryRead(s))
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))
	eq(uint64(92), r.GetBitPosition())

	eq(nil, r.TryError)
}

func TestCountWriter(t *testing.T) {
	for i := 0; i < 2; i++ {
		// 2 rounds, first use something that implements io.ByteWriter (*bytes.Buffer),
		// next testWriter which does not.
		var b interface {
			io.Writer
			Bytes() []byte
		}
		{
			buf := &bytes.Buffer{}
			b = buf
			if i > 0 {
				b = &testWriter{b: buf}
			}
		}

		w := NewCountWriter(b)

		expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0, 0xff, 0x80, 0x12, 0x34}

		eq, expEq := mighty.EqExpEq(t)

		eq(uint64(0), w.GetBufferBitSize())

		eq(nil, w.WriteByte(0xc1))
		eq(uint64(8), w.GetBufferBitSize())
		eq(nil, w.WriteBool(false))
		eq(uint64(9), w.GetBufferBitSize())
		eq(nil, w.WriteBits(0x3f, 6))
		eq(uint64(15), w.GetBufferBitSize())
		eq(nil, w.WriteBool(true))
		eq(uint64(16), w.GetBufferBitSize())
		eq(nil, w.WriteByte(0xac))
		eq(uint64(24), w.GetBufferBitSize())
		eq(nil, w.WriteBits(0x01, 1))
		eq(uint64(25), w.GetBufferBitSize())
		eq(nil, w.WriteBits(0x1248f, 20))
		eq(uint64(45), w.GetBufferBitSize())
		expEq(uint8(3))(w.Align())
		eq(uint64(48), w.GetBufferBitSize())
		expEq(2)(w.Write([]byte{0x01, 0x02}))
		eq(uint64(64), w.GetBufferBitSize())
		eq(nil, w.WriteBits(0x0f, 4))
		eq(uint64(68), w.GetBufferBitSize())
		expEq(2)(w.Write([]byte{0x80, 0x8f}))
		eq(uint64(84), w.GetBufferBitSize())
		expEq(uint8(4))(w.Align())
		eq(uint64(88), w.GetBufferBitSize())
		expEq(uint8(0))(w.Align())
		eq(uint64(88), w.GetBufferBitSize())
		eq(nil, w.WriteBits(0x01, 1))
		eq(uint64(89), w.GetBufferBitSize())
		eq(nil, w.WriteByte(0xff))
		eq(uint64(97), w.GetBufferBitSize())
		eq(uint8(7), w.TryAlign())
		eq(uint64(104), w.GetBufferBitSize())
		w.WriteBitsUnsafe(0x1234, 16)
		eq(uint64(120), w.GetBufferBitSize())
		eq(nil, w.Close())

		eq(true, bytes.Equal(b.Bytes(), expected))
	}
}

func TestCountWriterTry(t *testing.T) {
	for i := 0; i < 2; i++ {
		// 2 rounds, first use something that implements io.ByteWriter (*bytes.Buffer),
		// next testWriter which does not.
		var b interface {
			io.Writer
			Bytes() []byte
		}
		{
			buf := &bytes.Buffer{}
			b = buf
			if i > 0 {
				b = &testWriter{b: buf}
			}
		}

		w := NewCountWriter(b)

		expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0, 0xff, 0x80, 0x12, 0x34}

		eq := mighty.Eq(t)

		w.TryWriteByte(0xc1)
		eq(uint64(8), w.GetBufferBitSize())
		w.TryWriteBool(false)
		w.TryWriteBits(0x3f, 6)
		eq(uint64(15), w.GetBufferBitSize())
		w.TryWriteBool(true)
		eq(uint64(16), w.GetBufferBitSize())
		w.TryWriteByte(0xac)
		eq(uint64(24), w.GetBufferBitSize())
		w.TryWriteBits(0x01, 1)
		eq(uint64(25), w.GetBufferBitSize())
		w.TryWriteBits(0x1248f, 20)
		eq(uint64(45), w.GetBufferBitSize())
		eq(nil, w.TryError)

		eq(uint8(3), w.TryAlign())
		eq(nil, w.TryError)

		eq(2, w.TryWrite([]byte{0x01, 0x02}))
		eq(nil, w.TryError)

		w.TryWriteBits(0x0f, 4)
		eq(nil, w.TryError)

		eq(2, w.TryWrite([]byte{0x80, 0x8f}))
		eq(nil, w.TryError)

		eq(uint8(4), w.TryAlign())
		eq(nil, w.TryError)
		eq(uint8(0), w.TryAlign())
		eq(nil, w.TryError)
		w.TryWriteBits(0x01, 1)
		w.TryWriteByte(0xff)
		eq(nil, w.TryError)

		eq(uint8(7), w.TryAlign())
		w.TryWriteBitsUnsafe(0x1234, 16)

		eq(nil, w.Close())

		eq(true, bytes.Equal(b.Bytes(), expected))
	}
}

func TestCountReaderEOF(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))

	eq(uint64(0), r.GetBitPosition())
	b, err := r.ReadByte()
	eq(byte(1), b)
	eq(uint64(8), r.GetBitPosition())
	eq(nil, err)
	_, err = r.ReadByte()
	eq(io.EOF, err)
	_, err = r.ReadBool()
	eq(io.EOF, err)
	_, err = r.ReadBits(1)
	eq(io.EOF, err)
	n, err := r.Read(make([]byte, 2))
	eq(0, n)
	eq(io.EOF, err)
	eq(uint64(8), r.GetBitPosition())
}

func TestCountReaderTryEOF(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))

	b := r.TryReadByte()
	eq(byte(1), b)
	eq(nil, r.TryError)
	eq(uint64(8), r.GetBitPosition())
	_ = r.TryReadByte()
	eq(io.EOF, r.TryError)
	_ = r.TryReadBool()
	eq(io.EOF, r.TryError)
	_ = r.TryReadBits(1)
	eq(io.EOF, r.TryError)
	n := r.TryRead(make([]byte, 2))
	eq(0, n)
	eq(io.EOF, r.TryError)
	eq(uint64(8), r.GetBitPosition())
}

func TestCountReaderEOF2(t *testing.T) {
	eq, expEq := mighty.EqExpEq(t)

	var err error

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))
	_, err = r.ReadBits(17)
	eq(uint64(0), r.GetBitPosition())
	eq(io.EOF, err)

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	expEq(true)(r.ReadBool())
	eq(uint64(1), r.GetBitPosition())
	expEq(byte(0x82))(r.ReadByte())
	// readUnalignedByte resulting in EOF
	_, err = r.ReadByte()
	eq(io.EOF, err)
	eq(uint64(9), r.GetBitPosition())

	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	expEq(true)(r.ReadBool())
	got, err := r.Read(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, err)
	eq(uint64(9), r.GetBitPosition())
}

func TestCountReaderTryEOF2(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))
	_ = r.TryReadBits(17)
	eq(io.EOF, r.TryError)
	eq(uint64(0), r.GetBitPosition())

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	eq(true, r.TryReadBool())
	eq(nil, r.TryError)
	eq(uint64(1), r.GetBitPosition())
	eq(byte(0x82), r.TryReadByte())
	eq(nil, r.TryError)
	eq(uint64(9), r.GetBitPosition())
	// readUnalignedByte resulting in EOF
	_ = r.TryReadByte()
	eq(io.EOF, r.TryError)
	eq(uint64(9), r.GetBitPosition())

	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	eq(true, r.TryReadBool())
	eq(uint64(1), r.GetBitPosition())
	got := r.TryRead(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, r.TryError)
	eq(uint64(9), r.GetBitPosition())
}

func TestCountWriterError(t *testing.T) {
	eq, neq := mighty.EqNeq(t)

	w := NewCountWriter(&errWriter{1})
	eq(nil, w.WriteBool(true))
	eq(uint64(1), w.GetBufferBitSize())
	got, err := w.Write([]byte{0x01, 0x02})
	eq(1, got)
	neq(nil, err)
	neq(nil, w.Close())
	eq(uint64(9), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{0})
	neq(nil, w.WriteBits(0x00, 9))
	eq(uint64(0), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{1})
	neq(nil, w.WriteBits(0x00, 17))
	eq(uint64(0), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{})
	eq(nil, w.WriteBits(0x00, 7))
	neq(nil, w.WriteBool(false))
	eq(uint64(7), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{})
	eq(nil, w.WriteBool(true))
	_, err = w.Align()
	neq(nil, err)
	eq(uint64(1), w.GetBufferBitSize())
}

func TestCountWriterTryError(t *testing.T) {
	eq, neq := mighty.EqNeq(t)

	w := NewCountWriter(&errWriter{1})
	w.TryWriteBool(true)
	eq(nil, w.TryError)
	eq(uint64(1), w.GetBufferBitSize())
	got := w.TryWrite([]byte{0x01, 0x02})
	eq(1, got)
	neq(nil, w.TryError)
	neq(nil, w.Close())
	eq(uint64(9), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{0})
	w.TryWriteBits(0x00, 9)
	neq(nil, w.TryError)
	eq(uint64(0), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{1})
	w.TryWriteBits(0x00, 17)
	neq(nil, w.TryError)
	eq(uint64(0), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{})
	w.TryWriteBits(0x00, 7)
	eq(nil, w.TryError)
	eq(uint64(7), w.GetBufferBitSize())
	w.TryWriteBool(false)
	neq(nil, w.TryError)
	eq(uint64(7), w.GetBufferBitSize())

	w = NewCountWriter(&errWriter{})
	w.TryWriteBool(true)
	eq(nil, w.TryError)
	eq(uint64(1), w.GetBufferBitSize())
	_ = w.TryAlign()
	neq(nil, w.TryError)
	eq(uint64(1), w.GetBufferBitSize())
}

func TestCountedChain(t *testing.T) {
	eq, expEq := mighty.Eq(t), mighty.ExpEq(t)

	b := &bytes.Buffer{}
	w := NewCountWriter(b)

	rand.Seed(time.Now().UnixNano())

	expected := make([]uint64, 100000)
	bits := make([]byte, len(expected))
	expectedWriteSize := uint64(0)
	// Writing (generating)
	for i := range expected {
		expected[i] = uint64(rand.Int63())
		bits[i] = byte(1 + rand.Int31n(60))
		expected[i] &= uint64(1)<<bits[i] - 1
		w.WriteBits(expected[i], bits[i])
		expectedWriteSize += uint64(bits[i])
		eq(expectedWriteSize, w.GetBufferBitSize())
	}

	skipped, err := w.Align()
	eq(nil, err)
	eq(nil, w.Close())

	r := NewCountReader(bytes.NewBuffer(b.Bytes()))
	expectedReadSize := uint64(0)

	// Reading (verifying)
	for i, v := range expected {
		expEq(v)(r.ReadBits(bits[i]))
		expectedReadSize += uint64(bits[i])
		eq(expectedReadSize, r.GetBitPosition())
	}

	eq(r.GetBitPosition()+uint64(skipped), w.GetBufferBitSize())
}
