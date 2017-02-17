/*

Reader interface definition and implementation.

*/

package bitio

import (
	"bufio"
	"io"
)

// Reader is the bit reader interface.
type Reader interface {
	// Reader is an io.Reader
	io.Reader

	// Reader is also an io.ByteReader.
	// ReadByte reads the next 8 bits and returns them as a byte.
	io.ByteReader

	// ReadBits reads n bits and returns them as the lowest n bits of u.
	ReadBits(n byte) (u uint64, err error)

	// ReadBool reads the next bit, and returns true if it is 1.
	ReadBool() (b bool, err error)

	// Align aligns the bit stream to a byte boundary,
	// so next read will read/use data from the next byte.
	// Returns the number of unread / skipped bits.
	Align() (skipped byte)
}

// An io.Reader and io.ByteReader at the same time.
type readerAndByteReader interface {
	io.Reader
	io.ByteReader
}

// reader is the bit reader implementation.
type reader struct {
	in    readerAndByteReader
	cache byte // unread bits are stored here
	bits  byte // number of unread bits in cache
}

// NewReader returns a new Reader using the specified io.Reader as the input (source).
func NewReader(in io.Reader) Reader {
	var bin readerAndByteReader
	bin, ok := in.(readerAndByteReader)
	if !ok {
		bin = bufio.NewReader(in)
	}
	return &reader{in: bin}
}

// Read implements io.Reader.
func (r *reader) Read(p []byte) (n int, err error) {
	// r.bits will be the same after reading 8 bits, so we don't need to update that.
	if r.bits == 0 {
		return r.in.Read(p)
	}

	for ; n < len(p); n++ {
		if p[n], err = r.readUnalignedByte(); err != nil {
			return
		}
	}

	return
}

func (r *reader) ReadBits(n byte) (u uint64, err error) {
	// Some optimization, frequent cases
	if n < r.bits {
		// cache has all needed bits, and there are some extra which will be left in cache
		shift := r.bits - n
		u = uint64(r.cache >> shift)
		r.cache &= 1<<shift - 1
		r.bits = shift
		return
	}

	if n > r.bits {
		// all cache bits needed, and it's not even enough so more will be read
		if r.bits > 0 {
			u = uint64(r.cache)
			n -= r.bits
		}
		// Read whole bytes
		for n >= 8 {
			b, err2 := r.in.ReadByte()
			if err2 != nil {
				return 0, err2
			}
			u = u<<8 + uint64(b)
			n -= 8
		}
		// Read last fraction, if any
		if n > 0 {
			if r.cache, err = r.in.ReadByte(); err != nil {
				return 0, err
			}
			shift := 8 - n
			u = u<<n + uint64(r.cache>>shift)
			r.cache &= 1<<shift - 1
			r.bits = shift
		} else {
			r.bits = 0
		}
		return u, nil
	}

	// cache has exactly as many as needed
	r.bits = 0 // no need to clear cache, will be overridden on next read
	return uint64(r.cache), nil
}

// ReadByte implements io.ByteReader.
func (r *reader) ReadByte() (b byte, err error) {
	// r.bits will be the same after reading 8 bits, so we don't need to update that.
	if r.bits == 0 {
		return r.in.ReadByte()
	}
	return r.readUnalignedByte()
}

// readUnalignedByte reads the next 8 bits which are (may be) unaligned and returns them as a byte.
func (r *reader) readUnalignedByte() (b byte, err error) {
	// r.bits will be the same after reading 8 bits, so we don't need to update that.
	bits := r.bits
	b = r.cache << (8 - bits)
	r.cache, err = r.in.ReadByte()
	if err != nil {
		return 0, err
	}
	b |= r.cache >> bits
	r.cache &= 1<<bits - 1
	return
}

func (r *reader) ReadBool() (b bool, err error) {
	if r.bits == 0 {
		r.cache, err = r.in.ReadByte()
		if err != nil {
			return
		}
		b = (r.cache & 0x80) != 0
		r.cache, r.bits = r.cache&0x7f, 7
		return
	}

	r.bits--
	b = (r.cache & (1 << r.bits)) != 0
	r.cache &= 1<<r.bits - 1
	return
}

func (r *reader) Align() (skipped byte) {
	skipped = r.bits
	r.bits = 0 // no need to clear cache, will be overwritten on next read
	return
}
