package bitio

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/icza/mighty"
)

func TestReader(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewReader(bytes.NewBuffer(data))
	eq, expEq := mighty.EqExpEq(t)

	expEq(byte(3))(r.ReadByte())
	expEq(uint64(255))(r.ReadBits(8))

	expEq(uint64(0xc))(r.ReadBits(4))

	expEq(uint64(0xc1))(r.ReadBits(8))

	expEq(uint64(0xabcde))(r.ReadBits(20))

	expEq(true)(r.ReadBool())
	expEq(false)(r.ReadBool())

	eq(uint8(6), r.Align())

	s := make([]byte, 2)
	expEq(2)(r.Read(s))
	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))

	expEq(uint64(0xf))(r.ReadBits(4))

	expEq(2)(r.Read(s))
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))
}

func TestReaderTry(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewReader(bytes.NewBuffer(data))
	eq := mighty.Eq(t)

	eq(byte(3), r.TryReadByte())
	eq(uint64(255), r.TryReadBits(8))

	eq(uint64(0xc), r.TryReadBits(4))

	eq(uint64(0xc1), r.TryReadBits(8))

	eq(uint64(0xabcde), r.TryReadBits(20))

	eq(true, r.TryReadBool())
	eq(false, r.TryReadBool())

	eq(uint8(6), r.Align())

	s := make([]byte, 2)
	eq(2, r.TryRead(s))
	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))

	eq(uint64(0xf), r.TryReadBits(4))

	eq(2, r.TryRead(s))
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))

	eq(nil, r.TryError)
}

// testWriter that does not implement io.ByteWriter so we can test the
// behaviour of Writer when it creates an internal bufio.Writer.
type testWriter struct {
	b *bytes.Buffer
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return w.b.Write(p)
}

func (w *testWriter) Bytes() []byte {
	return w.b.Bytes()
}

func TestWriter(t *testing.T) {
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

		w := NewWriter(b)

		expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0, 0xff, 0x80, 0x12, 0x34}

		eq, expEq := mighty.EqExpEq(t)

		eq(nil, w.WriteByte(0xc1))
		eq(nil, w.WriteBool(false))
		eq(nil, w.WriteBits(0x3f, 6))
		eq(nil, w.WriteBool(true))
		eq(nil, w.WriteByte(0xac))
		eq(nil, w.WriteBits(0x01, 1))
		eq(nil, w.WriteBits(0x1248f, 20))

		expEq(uint8(3))(w.Align())

		expEq(2)(w.Write([]byte{0x01, 0x02}))

		eq(nil, w.WriteBits(0x0f, 4))

		expEq(2)(w.Write([]byte{0x80, 0x8f}))

		expEq(uint8(4))(w.Align())
		expEq(uint8(0))(w.Align())
		eq(nil, w.WriteBits(0x01, 1))
		eq(nil, w.WriteByte(0xff))

		eq(uint8(7), w.TryAlign())
		w.WriteBitsUnsafe(0x1234, 16)

		eq(nil, w.Close())

		eq(true, bytes.Equal(b.Bytes(), expected))
	}
}

func TestWriterTry(t *testing.T) {
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

		w := NewWriter(b)

		expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0, 0xff, 0x80, 0x12, 0x34}

		eq := mighty.Eq(t)

		w.TryWriteByte(0xc1)
		w.TryWriteBool(false)
		w.TryWriteBits(0x3f, 6)
		w.TryWriteBool(true)
		w.TryWriteByte(0xac)
		w.TryWriteBits(0x01, 1)
		w.TryWriteBits(0x1248f, 20)
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

func TestReaderEOF(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewReader(bytes.NewBuffer([]byte{0x01}))

	b, err := r.ReadByte()
	eq(byte(1), b)
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
}
func TestReaderUnalignedEOF(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewReader(bytes.NewBuffer([]byte{0xFF}))

	b, err := r.ReadBits(4)
	eq(byte(15), byte(b))
	eq(nil, err)
	b, err = r.ReadBits(5)
	eq(byte(15), byte(b))
	eq(io.EOF, err)
}

func TestReaderTryEOF(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewReader(bytes.NewBuffer([]byte{0x01}))

	b := r.TryReadByte()
	eq(byte(1), b)
	eq(nil, r.TryError)
	_ = r.TryReadByte()
	eq(io.EOF, r.TryError)
	_ = r.TryReadBool()
	eq(io.EOF, r.TryError)
	_ = r.TryReadBits(1)
	eq(io.EOF, r.TryError)
	n := r.TryRead(make([]byte, 2))
	eq(0, n)
	eq(io.EOF, r.TryError)
}

func TestReaderEOF2(t *testing.T) {
	eq, expEq := mighty.EqExpEq(t)

	var err error

	r := NewReader(bytes.NewBuffer([]byte{0x01}))
	_, err = r.ReadBits(17)
	eq(io.EOF, err)

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	expEq(true)(r.ReadBool())
	expEq(byte(0x82))(r.ReadByte())
	// readUnalignedByte resulting in EOF
	_, err = r.ReadByte()
	eq(io.EOF, err)

	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	expEq(true)(r.ReadBool())
	got, err := r.Read(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, err)
}

func TestReaderTryEOF2(t *testing.T) {
	eq := mighty.Eq(t)

	r := NewReader(bytes.NewBuffer([]byte{0x01}))
	_ = r.TryReadBits(17)
	eq(io.EOF, r.TryError)

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	eq(true, r.TryReadBool())
	eq(nil, r.TryError)
	eq(byte(0x82), r.TryReadByte())
	eq(nil, r.TryError)
	// readUnalignedByte resulting in EOF
	_ = r.TryReadByte()
	eq(io.EOF, r.TryError)

	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	eq(true, r.TryReadBool())
	got := r.TryRead(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, r.TryError)
}

type nonByteReaderWriter struct {
	io.Reader
	io.Writer
}

func TestNonByteReaderWriter(t *testing.T) {
	NewReader(nonByteReaderWriter{})
	NewWriter(nonByteReaderWriter{})
}

type errWriter struct {
	limit int
}

func (e *errWriter) WriteByte(c byte) error {
	if e.limit == 0 {
		return errors.New("Can't write more")
	}
	e.limit--
	return nil
}

func (e *errWriter) Write(p []byte) (n int, err error) {
	for i, v := range p {
		if err := e.WriteByte(v); err != nil {
			return i, err
		}
	}
	return len(p), nil
}

func TestWriterError(t *testing.T) {
	eq, neq := mighty.EqNeq(t)

	w := NewWriter(&errWriter{1})
	eq(nil, w.WriteBool(true))
	got, err := w.Write([]byte{0x01, 0x02})
	eq(1, got)
	neq(nil, err)
	neq(nil, w.Close())

	w = NewWriter(&errWriter{0})
	neq(nil, w.WriteBits(0x00, 9))

	w = NewWriter(&errWriter{1})
	neq(nil, w.WriteBits(0x00, 17))

	w = NewWriter(&errWriter{})
	eq(nil, w.WriteBits(0x00, 7))
	neq(nil, w.WriteBool(false))

	w = NewWriter(&errWriter{})
	eq(nil, w.WriteBool(true))
	_, err = w.Align()
	neq(nil, err)
}

func TestWriterTryError(t *testing.T) {
	eq, neq := mighty.EqNeq(t)

	w := NewWriter(&errWriter{1})
	w.TryWriteBool(true)
	eq(nil, w.TryError)
	got := w.TryWrite([]byte{0x01, 0x02})
	eq(1, got)
	neq(nil, w.TryError)
	neq(nil, w.Close())

	w = NewWriter(&errWriter{0})
	w.TryWriteBits(0x00, 9)
	neq(nil, w.TryError)

	w = NewWriter(&errWriter{1})
	w.TryWriteBits(0x00, 17)
	neq(nil, w.TryError)

	w = NewWriter(&errWriter{})
	w.TryWriteBits(0x00, 7)
	eq(nil, w.TryError)
	w.TryWriteBool(false)
	neq(nil, w.TryError)

	w = NewWriter(&errWriter{})
	w.TryWriteBool(true)
	eq(nil, w.TryError)
	_ = w.TryAlign()
	neq(nil, w.TryError)
}

func TestChain(t *testing.T) {
	eq, expEq := mighty.Eq(t), mighty.ExpEq(t)

	b := &bytes.Buffer{}
	w := NewWriter(b)

	rand.Seed(time.Now().UnixNano())

	expected := make([]uint64, 100000)
	bits := make([]byte, len(expected))

	// Writing (generating)
	for i := range expected {
		expected[i] = uint64(rand.Int63())
		bits[i] = byte(1 + rand.Int31n(60))
		expected[i] &= uint64(1)<<bits[i] - 1
		w.WriteBits(expected[i], bits[i])
	}

	eq(nil, w.Close())

	r := NewReader(bytes.NewBuffer(b.Bytes()))

	// Reading (verifying)
	for i, v := range expected {
		expEq(v)(r.ReadBits(bits[i]))
	}
}
