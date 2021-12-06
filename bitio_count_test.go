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

	eq(int64(0), r.BitsCount)

	expEq(byte(3))(r.ReadByte())
	eq(int64(8), r.BitsCount)

	expEq(uint64(255))(r.ReadBits(8))
	eq(int64(16), r.BitsCount)

	expEq(uint64(0xc))(r.ReadBits(4))
	eq(int64(20), r.BitsCount)

	expEq(uint64(0xc1))(r.ReadBits(8))
	eq(int64(28), r.BitsCount)

	expEq(uint64(0xabcde))(r.ReadBits(20))
	eq(int64(48), r.BitsCount)

	expEq(true)(r.ReadBool())
	eq(int64(49), r.BitsCount)

	expEq(false)(r.ReadBool())
	eq(int64(50), r.BitsCount)

	eq(uint8(6), r.Align())
	eq(int64(56), r.BitsCount)

	s := make([]byte, 2)
	expEq(2)(r.Read(s))
	eq(int64(72), r.BitsCount)

	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))

	expEq(uint64(0xf))(r.ReadBits(4))
	eq(int64(76), r.BitsCount)

	expEq(2)(r.Read(s))
	eq(int64(92), r.BitsCount)
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))
}

func TestCountReaderTry(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewCountReader(bytes.NewBuffer(data))
	eq := mighty.Eq(t)

	eq(int64(0), r.BitsCount)

	eq(byte(3), r.TryReadByte())
	eq(int64(8), r.BitsCount)

	eq(uint64(255), r.TryReadBits(8))
	eq(int64(16), r.BitsCount)

	eq(uint64(0xc), r.TryReadBits(4))
	eq(int64(20), r.BitsCount)

	eq(uint64(0xc1), r.TryReadBits(8))
	eq(int64(28), r.BitsCount)

	eq(uint64(0xabcde), r.TryReadBits(20))
	eq(int64(48), r.BitsCount)

	eq(true, r.TryReadBool())
	eq(false, r.TryReadBool())
	eq(int64(50), r.BitsCount)

	eq(uint8(6), r.Align())
	eq(int64(56), r.BitsCount)

	s := make([]byte, 2)
	eq(2, r.TryRead(s))
	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))
	eq(int64(72), r.BitsCount)

	eq(uint64(0xf), r.TryReadBits(4))
	eq(int64(76), r.BitsCount)

	eq(2, r.TryRead(s))
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))
	eq(int64(92), r.BitsCount)

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

		eq(int64(0), w.BitsCount)

		eq(nil, w.WriteByte(0xc1))
		eq(int64(8), w.BitsCount)
		eq(nil, w.WriteBool(false))
		eq(int64(9), w.BitsCount)
		eq(nil, w.WriteBits(0x3f, 6))
		eq(int64(15), w.BitsCount)
		eq(nil, w.WriteBool(true))
		eq(int64(16), w.BitsCount)
		eq(nil, w.WriteByte(0xac))
		eq(int64(24), w.BitsCount)
		eq(nil, w.WriteBits(0x01, 1))
		eq(int64(25), w.BitsCount)
		eq(nil, w.WriteBits(0x1248f, 20))
		eq(int64(45), w.BitsCount)
		expEq(uint8(3))(w.Align())
		eq(int64(48), w.BitsCount)
		expEq(2)(w.Write([]byte{0x01, 0x02}))
		eq(int64(64), w.BitsCount)
		eq(nil, w.WriteBits(0x0f, 4))
		eq(int64(68), w.BitsCount)
		expEq(2)(w.Write([]byte{0x80, 0x8f}))
		eq(int64(84), w.BitsCount)
		expEq(uint8(4))(w.Align())
		eq(int64(88), w.BitsCount)
		expEq(uint8(0))(w.Align())
		eq(int64(88), w.BitsCount)
		eq(nil, w.WriteBits(0x01, 1))
		eq(int64(89), w.BitsCount)
		eq(nil, w.WriteByte(0xff))
		eq(int64(97), w.BitsCount)
		eq(uint8(7), w.TryAlign())
		eq(int64(104), w.BitsCount)
		w.WriteBitsUnsafe(0x1234, 16)
		eq(int64(120), w.BitsCount)
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
		eq(int64(8), w.BitsCount)
		w.TryWriteBool(false)
		w.TryWriteBits(0x3f, 6)
		eq(int64(15), w.BitsCount)
		w.TryWriteBool(true)
		eq(int64(16), w.BitsCount)
		w.TryWriteByte(0xac)
		eq(int64(24), w.BitsCount)
		w.TryWriteBits(0x01, 1)
		eq(int64(25), w.BitsCount)
		w.TryWriteBits(0x1248f, 20)
		eq(int64(45), w.BitsCount)
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

	eq(int64(0), r.BitsCount)
	b, err := r.ReadByte()
	eq(byte(1), b)
	eq(int64(8), r.BitsCount)
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
	eq(int64(8), r.BitsCount)
}

func TestCountReaderTryEOF(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))

	b := r.TryReadByte()
	eq(byte(1), b)
	eq(nil, r.TryError)
	eq(int64(8), r.BitsCount)
	_ = r.TryReadByte()
	eq(io.EOF, r.TryError)
	_ = r.TryReadBool()
	eq(io.EOF, r.TryError)
	_ = r.TryReadBits(1)
	eq(io.EOF, r.TryError)
	n := r.TryRead(make([]byte, 2))
	eq(0, n)
	eq(io.EOF, r.TryError)
	eq(int64(8), r.BitsCount)
}

func TestCountReaderEOF2(t *testing.T) {
	eq, expEq := mighty.EqExpEq(t)

	var err error

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))
	_, err = r.ReadBits(17)
	eq(int64(0), r.BitsCount)
	eq(io.EOF, err)

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	expEq(true)(r.ReadBool())
	eq(int64(1), r.BitsCount)
	expEq(byte(0x82))(r.ReadByte())
	// readUnalignedByte resulting in EOF
	_, err = r.ReadByte()
	eq(io.EOF, err)
	eq(int64(9), r.BitsCount)

	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	expEq(true)(r.ReadBool())
	got, err := r.Read(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, err)
	eq(int64(9), r.BitsCount)
}

func TestCountReaderTryEOF2(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewCountReader(bytes.NewBuffer([]byte{0x01}))
	_ = r.TryReadBits(17)
	eq(io.EOF, r.TryError)
	eq(int64(0), r.BitsCount)

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	eq(true, r.TryReadBool())
	eq(nil, r.TryError)
	eq(int64(1), r.BitsCount)
	eq(byte(0x82), r.TryReadByte())
	eq(nil, r.TryError)
	eq(int64(9), r.BitsCount)
	// readUnalignedByte resulting in EOF
	_ = r.TryReadByte()
	eq(io.EOF, r.TryError)
	eq(int64(9), r.BitsCount)

	r = NewCountReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	eq(true, r.TryReadBool())
	eq(int64(1), r.BitsCount)
	got := r.TryRead(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, r.TryError)
	eq(int64(9), r.BitsCount)
}

func TestCountWriterError(t *testing.T) {
	eq, neq := mighty.EqNeq(t)

	w := NewCountWriter(&errWriter{1})
	eq(nil, w.WriteBool(true))
	eq(int64(1), w.BitsCount)
	got, err := w.Write([]byte{0x01, 0x02})
	eq(1, got)
	neq(nil, err)
	neq(nil, w.Close())
	eq(int64(9), w.BitsCount)

	w = NewCountWriter(&errWriter{0})
	neq(nil, w.WriteBits(0x00, 9))
	eq(int64(0), w.BitsCount)

	w = NewCountWriter(&errWriter{1})
	neq(nil, w.WriteBits(0x00, 17))
	eq(int64(0), w.BitsCount)

	w = NewCountWriter(&errWriter{})
	eq(nil, w.WriteBits(0x00, 7))
	neq(nil, w.WriteBool(false))
	eq(int64(7), w.BitsCount)

	w = NewCountWriter(&errWriter{})
	eq(nil, w.WriteBool(true))
	_, err = w.Align()
	neq(nil, err)
	eq(int64(1), w.BitsCount)
}

func TestCountWriterTryError(t *testing.T) {
	eq, neq := mighty.EqNeq(t)

	w := NewCountWriter(&errWriter{1})
	w.TryWriteBool(true)
	eq(nil, w.TryError)
	eq(int64(1), w.BitsCount)
	got := w.TryWrite([]byte{0x01, 0x02})
	eq(1, got)
	neq(nil, w.TryError)
	neq(nil, w.Close())
	eq(int64(9), w.BitsCount)

	w = NewCountWriter(&errWriter{0})
	w.TryWriteBits(0x00, 9)
	neq(nil, w.TryError)
	eq(int64(0), w.BitsCount)

	w = NewCountWriter(&errWriter{1})
	w.TryWriteBits(0x00, 17)
	neq(nil, w.TryError)
	eq(int64(0), w.BitsCount)

	w = NewCountWriter(&errWriter{})
	w.TryWriteBits(0x00, 7)
	eq(nil, w.TryError)
	eq(int64(7), w.BitsCount)
	w.TryWriteBool(false)
	neq(nil, w.TryError)
	eq(int64(7), w.BitsCount)

	w = NewCountWriter(&errWriter{})
	w.TryWriteBool(true)
	eq(nil, w.TryError)
	eq(int64(1), w.BitsCount)
	_ = w.TryAlign()
	neq(nil, w.TryError)
	eq(int64(1), w.BitsCount)
}

func TestCountedChain(t *testing.T) {
	eq, expEq := mighty.Eq(t), mighty.ExpEq(t)

	b := &bytes.Buffer{}
	w := NewCountWriter(b)

	rand.Seed(time.Now().UnixNano())

	expected := make([]uint64, 100000)
	bits := make([]byte, len(expected))
	expectedWriteSize := int64(0)
	// Writing (generating)
	for i := range expected {
		expected[i] = uint64(rand.Int63())
		bits[i] = byte(1 + rand.Int31n(60))
		expected[i] &= uint64(1)<<bits[i] - 1
		w.WriteBits(expected[i], bits[i])
		expectedWriteSize += int64(bits[i])
		eq(expectedWriteSize, w.BitsCount)
	}

	skipped, err := w.Align()
	eq(nil, err)
	eq(nil, w.Close())

	r := NewCountReader(bytes.NewBuffer(b.Bytes()))
	expectedReadSize := int64(0)

	// Reading (verifying)
	for i, v := range expected {
		expEq(v)(r.ReadBits(bits[i]))
		expectedReadSize += int64(bits[i])
		eq(expectedReadSize, r.BitsCount)
	}

	eq(r.BitsCount+int64(skipped), w.BitsCount)
}
