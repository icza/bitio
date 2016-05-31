/*

Package bitio provides a highly optimized bit-level Reader and Writer.

Both Reader and Writer provide highly optimized methods for reading / writing
1 bit of information in the form of a bool value: Reader.ReadBool() and Writer.WriteBool().
These make this package ideal for compression algorithms that use Huffman coding for example,
where decision whether to step left or right in the Huffman tree is the most frequent operation.

Both Reader and Writer give a bit-view of the underlying io.Reader and io.Writer, but they also provide
an io.Reader and io.Writer view at the same time. This means you can also use the Reader.Read() and
Writer.Write() methods to read and write slices of bytes. These will give you best performance if
the underlying io.Reader and io.Writer are aligned to a byte boundary (else all the individual bytes
are assembled from / spread to multiple bytes). You can ensure byte boundary by calling the Align()
method of Reader or Writer.

*/
package bitio
