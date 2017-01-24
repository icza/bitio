# bitio

[![Build Status](https://travis-ci.org/icza/bitio.svg?branch=master)](https://travis-ci.org/icza/bitio)
[![GoDoc](https://godoc.org/github.com/icza/bitio?status.svg)](https://godoc.org/github.com/icza/bitio)
[![Go Report Card](https://goreportcard.com/badge/github.com/icza/bitio)](https://goreportcard.com/report/github.com/icza/bitio)
[![codecov](https://codecov.io/gh/icza/bitio/branch/master/graph/badge.svg)](https://codecov.io/gh/icza/bitio)

Package `bitio` provides a highly optimized bit-level `Reader` and `Writer` for Go.

You can use `Reader.ReadBits()` to read arbitrary number of bits from an `io.Reader` and return it as an `uint64`,
and `Writer.WriteBits()` to write arbitrary number of bits of an `uint64` value to an `io.Writer`.

Both `Reader` and `Writer` also provide highly optimized methods for reading / writing
1 bit of information in the form of a `bool` value: `Reader.ReadBool()` and `Writer.WriteBool()`.
These make this package ideal for compression algorithms that use [Huffman coding](https://en.wikipedia.org/wiki/Huffman_coding) for example,
where decision whether to step left or right in the Huffman tree is the most frequent operation.

`Reader` and `Writer` give a _bit-level_ view  of the underlying `io.Reader` and `io.Writer`, but they also
provide a _byte-level_ view (`io.Reader` and `io.Writer`) at the same time. This means you can also use
the `Reader.Read()` and `Writer.Write()` methods to read and write slices of bytes. These will give
you best performance if the underlying `io.Reader` and `io.Writer` are aligned to a byte boundary
(else all the individual bytes are assembled from / spread to multiple bytes). You can ensure
byte boundary by calling the `Align()` method of `Reader` and `Writer`.

### Bit order

The more general highest-bits-first order is used. So for example if the input provides the bytes `0x8f` and `0x55`:

    HEXA    8    f     5    5
    BINARY  1100 1111  0101 0101
            aaaa bbbc  ccdd dddd

Then ReadBits will return the following values:

    r := NewReader(bytes.NewBuffer([]byte{0x8f, 0x55}))
    a, err := r.ReadBits(4) //   1100 = 0x08
    b, err := r.ReadBits(3) //    111 = 0x07
    c, err := r.ReadBits(3) //    101 = 0x05
    d, err := r.ReadBits(6) // 010101 = 0x15

Writing the above values would result in the same sequence of bytes:

    b := &bytes.Buffer{}
    w := NewWriter(b)
    err := w.WriteBits(0x08, 4)
    err = w.WriteBits(0x07, 3)
    err = w.WriteBits(0x05, 3)
    err = w.WriteBits(0x15, 6)
    err = w.Close()
    // b will hold the bytes: 0x8f and 0x55
