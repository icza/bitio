/*

Writer interface definition and implementation.

*/

package bitio

import (
	"bufio"
	"io"
)

// Writer is the bit writer interface.
// Must be closed in order to flush cached data.
// If you can't or don't want to close it, flushing data can also be forced
// by calling Align().
type Writer interface {
	// Writer is an io.Writer and io.Closer.
	// Close closes the bit writer, writes out cached bits.
	// It does not close the underlying io.Writer.
	io.WriteCloser

	// Writer is also an io.ByteWriter.
	// WriteByte writes 8 bits.
	io.ByteWriter

	// WriteBits writes out the n lowest bits of r.
	// r cannot have bits set at positions higher than n-1 (zero indexed).
	WriteBits(r uint64, n byte) (err error)

	// WriteBool writes one bit: 1 if param is true, 0 otherwise.
	WriteBool(b bool) (err error)

	// Align aligns the bit stream to a byte boundary,
	// so next write will start/go into a new byte.
	// If there are cached bits, they are first written to the output.
	// Returns the number of skipped (unset but still written) bits.
	Align() (skipped byte, err error)
}

// An io.Writer and io.ByteWriter at the same time.
type writerAndByteWriter interface {
	io.Writer
	io.ByteWriter
}

// writer is the bit writer implementation.
type writer struct {
	out       writerAndByteWriter
	wrapperbw *bufio.Writer // wrapper bufio.Writer if the target does not implement io.ByteWriter
	cache     byte          // unwritten bits are stored here
	bits      byte          // number of unwritten bits in cache
}

// NewWriter returns a new Writer using the specified io.Writer as the output.
func NewWriter(out io.Writer) Writer {
	w := &writer{}
	var ok bool
	w.out, ok = out.(writerAndByteWriter)
	if !ok {
		w.wrapperbw = bufio.NewWriter(out)
		w.out = w.wrapperbw
	}
	return w
}

// Write implements io.Writer.
func (w *writer) Write(p []byte) (n int, err error) {
	// w.bits will be the same after writing 8 bits, so we don't need to update that.
	if w.bits == 0 {
		return w.out.Write(p)
	}

	for i, b := range p {
		if err = w.writeUnalignedByte(b); err != nil {
			return i, err
		}
	}

	return len(p), nil
}

func (w *writer) WriteBits(r uint64, n byte) (err error) {
	// Some optimization, frequent cases
	newbits := w.bits + n
	if newbits < 8 {
		// r fits into cache, no write will occur to out
		w.cache |= byte(r) << (8 - newbits)
		w.bits = newbits
		return nil
	}

	if newbits > 8 {
		// cache will be filled, and there will be more bits to write
		// "Fill cache" and write it out
		free := 8 - w.bits
		err = w.out.WriteByte(w.cache | byte(r>>(n-free)))
		if err != nil {
			return
		}
		n -= free
		// write out whole bytes
		for n >= 8 {
			n -= 8
			// No need to mask r, converting to byte will mask out higher bits
			err = w.out.WriteByte(byte(r >> n))
			if err != nil {
				return
			}
		}
		// Put remaining into cache
		if n > 0 {
			// Note: n < 8 (in case of n=8, 1<<n would overflow byte)
			w.cache, w.bits = (byte(r)&((1<<n)-1))<<(8-n), n
		} else {
			w.cache, w.bits = 0, 0
		}
		return nil
	}

	// cache will be filled exactly with the bits to be written
	bb := w.cache | byte(r)
	w.cache, w.bits = 0, 0
	return w.out.WriteByte(bb)
}

// WriteByte implements io.ByteWriter.
func (w *writer) WriteByte(b byte) (err error) {
	// w.bits will be the same after writing 8 bits, so we don't need to update that.
	if w.bits == 0 {
		return w.out.WriteByte(b)
	}
	return w.writeUnalignedByte(b)
}

// writeUnalignedByte writes 8 bits which are (may be) unaligned.
func (w *writer) writeUnalignedByte(b byte) (err error) {
	// w.bits will be the same after writing 8 bits, so we don't need to update that.
	bits := w.bits
	err = w.out.WriteByte(w.cache | b>>bits)
	if err != nil {
		return
	}
	w.cache = (b & (1<<bits - 1)) << (8 - bits)
	return
}

func (w *writer) WriteBool(b bool) (err error) {
	if w.bits == 7 {
		if b {
			err = w.out.WriteByte(w.cache | 1)
		} else {
			err = w.out.WriteByte(w.cache)
		}
		if err != nil {
			return
		}
		w.cache, w.bits = 0, 0
		return nil
	}

	w.bits++
	if b {
		w.cache |= 1 << (8 - w.bits)
	}
	return nil
}

func (w *writer) Align() (skipped byte, err error) {
	if w.bits > 0 {
		if err = w.out.WriteByte(w.cache); err != nil {
			return
		}

		skipped = 8 - w.bits
		w.cache, w.bits = 0, 0
	}
	if w.wrapperbw != nil {
		err = w.wrapperbw.Flush()
	}
	return
}

// Close implements io.Closer.
func (w *writer) Close() (err error) {
	// Make sure cached bits are flushed:
	if _, err = w.Align(); err != nil {
		return
	}

	return nil
}
