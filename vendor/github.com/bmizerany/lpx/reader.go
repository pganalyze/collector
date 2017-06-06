package lpx

import (
	"bytes"
	"io"
	"strconv"
)

type BytesReader interface {
	io.Reader
	ReadBytes(delim byte) (line []byte, err error)
}

// A Header represents a single header in a logplex entry. All fields are
// popluated.
type Header struct {
	PrivalVersion []byte
	Time          []byte
	Hostname      []byte
	Name          []byte
	Procid        []byte
	Msgid         []byte
}

// A Reader provides sequential access to logplex packages. The Next method
// advances to the next entry (including the first), and then can be treated
// as an io.Reader to access the packages payload.
type Reader struct {
	r     BytesReader
	b     io.Reader
	hdr   *Header
	err   error
	n     int64
	bytes []byte
}

// NewReader creates a new Reader reading from r.
func NewReader(r BytesReader) *Reader {
	return &Reader{r: r, hdr: new(Header)}
}

// Next advances to the next entry in the stream.
func (r *Reader) Next() bool {
	if r.err != nil {
		return false
	}

	// length
	var l []byte
	r.field(&l) // message length
	if r.err != nil {
		return false
	}

	r.n, r.err = strconv.ParseInt(string(l), 10, 64)
	if r.err != nil {
		return false
	}

	// header fields
	r.field(&r.hdr.PrivalVersion) // PRI/VERSION
	r.field(&r.hdr.Time)          // TIMESTAMP
	r.field(&r.hdr.Hostname)      // HOSTNAME
	r.field(&r.hdr.Name)          // APP-NAME
	r.field(&r.hdr.Procid)        // PROCID
	r.field(&r.hdr.Msgid)         // MSGID
	if r.err != nil {
		return false
	}

	// payload
	if r.n > 0 {
		r.bytes = make([]byte, r.n)
		_, r.err = io.ReadFull(r.r, r.bytes)
		if r.err != nil {
			if r.err == io.EOF {
				r.err = io.ErrUnexpectedEOF
			}
			return false
		}
	} else {
		r.bytes = nil
	}
	return true
}

// Bytes returns the message body.
func (r *Reader) Bytes() []byte {
	return r.bytes
}

// Header returns the current entries decoded header.
func (r *Reader) Header() *Header {
	return r.hdr
}

// Err returns the first non-EOF error that was encountered by the Reader.
func (r *Reader) Err() error {
	if r.err == io.EOF {
		return nil
	}
	return r.err
}

func (r *Reader) field(b *[]byte) {
	g, err := r.r.ReadBytes(' ')
	if err != nil {
		r.err = err
		return
	}
	r.n -= int64(len(g))
	if b == nil {
		return
	}
	*b = bytes.TrimRight(g, " ")
}
