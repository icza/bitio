package bitio

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	r := NewReader(bytes.NewBuffer(data))

	var nExp interface{}
	check := func(n interface{}, err error) {
		if n != nExp || err != nil {
			t.Errorf("Got %x, want %x, error: %v", n, nExp, err)
		}
	}

	nExp = byte(3)
	check(r.ReadByte())
	nExp = uint64(255)
	check(r.ReadBits(8))

	nExp = uint64(0xc)
	check(r.ReadBits(4))

	nExp = uint64(0xc1)
	check(r.ReadBits(8))

	nExp = uint64(0xabcde)
	check(r.ReadBits(20))

	if b, err := r.ReadBool(); !b || err != nil {
		t.Errorf("Got %v, want %v, error: %v", b, false, err)
	}
	if b, err := r.ReadBool(); b || err != nil {
		t.Errorf("Got %v, want %v, error: %v", b, true, err)
	}

	if n := r.Align(); n != 6 {
		t.Errorf("Got %v, want %v", n, 6)
	}

	s := make([]byte, 2)
	if n, err := r.Read(s); n != 2 || err != nil || !bytes.Equal(s, []byte{0x01, 0x02}) {
		t.Errorf("Got %v, want %v, error: %v", s, []byte{0x01, 0x02}, err)
	}

	if i, err := r.ReadBits(4); i != 0xf || err != nil {
		t.Errorf("Got %x, want %x, error: %v", i, 0xf, err)
	}

	if n, err := r.Read(s); n != 2 || err != nil || !bytes.Equal(s, []byte{0x80, 0x8f}) {
		t.Errorf("Got %v, want %v, error: %v", s, []byte{0x80, 0x8f}, err)
	}
}

func TestWriter(t *testing.T) {
	b := &bytes.Buffer{}

	w := NewWriter(b)

	expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0, 0xff, 0x80}

	errs := []error{}
	errs = append(errs, w.WriteByte(0xc1))
	errs = append(errs, w.WriteBool(false))
	errs = append(errs, w.WriteBits(0x3f, 6))
	errs = append(errs, w.WriteBool(true))
	errs = append(errs, w.WriteByte(0xac))
	errs = append(errs, w.WriteBits(0x01, 1))
	errs = append(errs, w.WriteBits(0x1248f, 20))

	var nExp interface{}
	check := func(n interface{}, err error) {
		if n != nExp || err != nil {
			t.Errorf("Got %x, want %x, error: %v", n, nExp, err)
		}
	}

	nExp = byte(3)
	check(w.Align())

	nExp = int(2)
	check(w.Write([]byte{0x01, 0x02}))

	errs = append(errs, w.WriteBits(0x0f, 4))

	check(w.Write([]byte{0x80, 0x8f}))

	nExp = byte(4)
	check(w.Align())
	nExp = byte(0)
	check(w.Align())
	if err := w.WriteBits(0x01, 1); err != nil {
		t.Error("Got error:", err)
	}
	if err := w.WriteByte(0xff); err != nil {
		t.Error("Got error:", err)
	}

	errs = append(errs, w.Close())

	for _, v := range errs {
		if v != nil {
			t.Error("Got error:", v)
		}
	}

	if !bytes.Equal(b.Bytes(), expected) {
		t.Errorf("Got: %x, want: %x", b.Bytes(), expected)
	}
}

func TestReaderEOF(t *testing.T) {
	r := NewReader(bytes.NewBuffer([]byte{0x01}))

	if b, err := r.ReadByte(); b != 1 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", b, 1, err)
	}
	if _, err := r.ReadByte(); err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}
	if _, err := r.ReadBool(); err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}
	if _, err := r.ReadBits(1); err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}
	if n, err := r.Read(make([]byte, 2)); n != 0 || err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}
}

func TestReaderEOF2(t *testing.T) {
	r := NewReader(bytes.NewBuffer([]byte{0x01}))
	if _, err := r.ReadBits(17); err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}

	// Byte spreading byte boundary (readUnalignedByte)
	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	if b, err := r.ReadBool(); !b || err != nil {
		t.Errorf("Got %v, want %v, error: %v", b, false, err)
	}
	if b, err := r.ReadByte(); b != 0x82 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", b, 0x82, err)
	}
	// readUnalignedByte resulting in EOF
	if _, err := r.ReadByte(); err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}

	r = NewReader(bytes.NewBuffer([]byte{0xc1, 0x01}))
	if b, err := r.ReadBool(); !b || err != nil {
		t.Errorf("Got %v, want %v, error: %v", b, false, err)
	}
	if n, err := r.Read(make([]byte, 2)); n != 1 || err != io.EOF {
		t.Errorf("Got %v, want %v", err, io.EOF)
	}
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
	w := NewWriter(&errWriter{1})
	if err := w.WriteBool(true); err != nil {
		t.Error("Got error:", err)
	}
	if n, err := w.Write([]byte{0x01, 0x02}); n != 1 || err == nil {
		t.Errorf("Got %x, want %x, error: %v", n, 2, err)
	}
	if err := w.Close(); err == nil {
		t.Error("Got no error:", err)
	}

	w = NewWriter(&errWriter{0})
	if err := w.WriteBits(0x00, 9); err == nil {
		t.Error("Got no error:", err)
	}

	w = NewWriter(&errWriter{1})
	if err := w.WriteBits(0x00, 17); err == nil {
		t.Error("Got no error:", err)
	}

	w = NewWriter(&errWriter{})
	if err := w.WriteBits(0x00, 7); err != nil {
		t.Error("Got error:", err)
	}
	if err := w.WriteBool(false); err == nil {
		t.Error("Got no error:", err)
	}

	w = NewWriter(&errWriter{})
	if err := w.WriteBool(true); err != nil {
		t.Error("Got error:", err)
	}
	if _, err := w.Align(); err == nil {
		t.Error("Got no error:", err)
	}

	w = NewWriter(&errCloser{})
	if err := w.Close(); err == nil {
		t.Error("Got no error:", err)
	}
}

func TestChain(t *testing.T) {
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
	if err := w.Close(); err != nil {
		t.Error("Got error:", err)
	}

	r := NewReader(bytes.NewBuffer(b.Bytes()))

	// Reading (verifying)
	for i, v := range expected {
		if u, err := r.ReadBits(bits[i]); u != v || err != nil {
			t.Errorf("Idx: %d, Got: %x, want: %x, bits: %d, error: %v", i, u, v, bits[i], err)
		}
	}
}
