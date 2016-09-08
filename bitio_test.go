package bitio

import (
	"bytes"
	"errors"
	"github.com/icza/mighty"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewReader(bytes.NewBuffer(data))

	eq := mighty.Eq(t)
	var exp interface{}
	check := func(got interface{}, err error) {
		eq(exp, got, err)
	}

	exp = byte(3)
	check(r.ReadByte())
	exp = uint64(255)
	check(r.ReadBits(8))

	exp = uint64(0xc)
	check(r.ReadBits(4))

	exp = uint64(0xc1)
	check(r.ReadBits(8))

	exp = uint64(0xabcde)
	check(r.ReadBits(20))

	exp = true
	check(r.ReadBool())
	exp = false
	check(r.ReadBool())

	eq(byte(6), r.Align())

	s := make([]byte, 2)
	exp = 2
	check(r.Read(s))
	eq(true, bytes.Equal(s, []byte{0x01, 0x02}))

	exp = uint64(0xf)
	check(r.ReadBits(4))

	exp = 2
	check(r.Read(s))
	eq(true, bytes.Equal(s, []byte{0x80, 0x8f}))
}

func TestWriter(t *testing.T) {
	b := &bytes.Buffer{}

	w := NewWriter(b)

	expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0, 0xff, 0x80}

	eq := mighty.Eq(t)

	eq(nil, w.WriteByte(0xc1))
	eq(nil, w.WriteBool(false))
	eq(nil, w.WriteBits(0x3f, 6))
	eq(nil, w.WriteBool(true))
	eq(nil, w.WriteByte(0xac))
	eq(nil, w.WriteBits(0x01, 1))
	eq(nil, w.WriteBits(0x1248f, 20))

	var exp interface{}
	check := func(got interface{}, err error) {
		eq(exp, got, err)
	}

	exp = byte(3)
	check(w.Align())

	exp = int(2)
	check(w.Write([]byte{0x01, 0x02}))

	eq(nil, w.WriteBits(0x0f, 4))

	check(w.Write([]byte{0x80, 0x8f}))

	exp = byte(4)
	check(w.Align())
	exp = byte(0)
	check(w.Align())
	eq(nil, w.WriteBits(0x01, 1))
	eq(nil, w.WriteByte(0xff))

	eq(nil, w.Close())

	eq(true, bytes.Equal(b.Bytes(), expected))
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

func TestReaderEOF2(t *testing.T) {
	eq := mighty.Eq(t)

	var exp interface{}
	var err error

	check := func(got interface{}, err error) {
		eq(exp, got, err)
	}

	r := NewReader(bytes.NewBuffer([]byte{0x01}))
	_, err = r.ReadBits(17)
	eq(io.EOF, err)

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	exp = true
	check(r.ReadBool())
	exp = byte(0x82)
	check(r.ReadByte())
	// readUnalignedByte resulting in EOF
	_, err = r.ReadByte()
	eq(io.EOF, err)

	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	exp = true
	check(r.ReadBool())
	got, err := r.Read(make([]byte, 2))
	eq(1, got)
	eq(io.EOF, err)
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
		return errors.New("Can't write more!")
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

type errCloser struct {
	errWriter
}

func (e *errCloser) Close() error {
	return errors.New("Obliged not to close!")
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

	w = NewWriter(&errCloser{})
	neq(nil, w.Close())
}

func TestChain(t *testing.T) {
	eq := mighty.Eq(t)

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
		u, err := r.ReadBits(bits[i])
		eq(v, u, err)
	}
}
