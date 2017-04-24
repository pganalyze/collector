package output

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"strconv"

	"github.com/aws/aws-sdk-go/service/s3/s3crypto"
)

// hashReader is used for calculating SHA256 when following the sigv4 specification.
// Additionally this used for calculating the unencrypted MD5.
type hashReader interface {
	GetValue() []byte
	GetContentLength() int64
}

type sha256Writer struct {
	sha256 []byte
	hash   hash.Hash
	out    io.Writer
}

func newSHA256Writer(f io.Writer) *sha256Writer {
	return &sha256Writer{hash: sha256.New(), out: f}
}
func (r *sha256Writer) Write(b []byte) (int, error) {
	r.hash.Write(b)
	return r.out.Write(b)
}

func (r *sha256Writer) GetValue() []byte {
	return r.hash.Sum(nil)
}

type md5Reader struct {
	contentLength int64
	hash          hash.Hash
	body          io.Reader
}

func newMD5Reader(body io.Reader) *md5Reader {
	return &md5Reader{hash: md5.New(), body: body}
}

func (w *md5Reader) Read(b []byte) (int, error) {
	n, err := w.body.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	}
	w.contentLength += int64(n)
	w.hash.Write(b[:n])
	return n, err
}

func (w *md5Reader) GetValue() []byte {
	return w.hash.Sum(nil)
}

func (w *md5Reader) GetContentLength() int64 {
	return w.contentLength
}

// ---

func encodeMeta(reader hashReader, cd s3crypto.CipherData) (s3crypto.Envelope, error) {
	iv := base64.StdEncoding.EncodeToString(cd.IV)
	key := base64.StdEncoding.EncodeToString(cd.EncryptedKey)

	md5 := reader.GetValue()
	contentLength := reader.GetContentLength()

	md5Str := base64.StdEncoding.EncodeToString(md5)
	matdesc, err := json.Marshal(&cd.MaterialDescription)
	if err != nil {
		return s3crypto.Envelope{}, err
	}

	return s3crypto.Envelope{
		CipherKey:             key,
		IV:                    iv,
		MatDesc:               string(matdesc),
		WrapAlg:               cd.WrapAlgorithm,
		CEKAlg:                cd.CEKAlgorithm,
		TagLen:                cd.TagLength,
		UnencryptedMD5:        md5Str,
		UnencryptedContentLen: strconv.FormatInt(contentLength, 10),
	}, nil
}

// ---

type bytesReadWriteSeeker struct {
	buf []byte
	i   int64
}

// Copied from Go stdlib bytes.Reader
func (ws *bytesReadWriteSeeker) Read(b []byte) (int, error) {
	if ws.i >= int64(len(ws.buf)) {
		return 0, io.EOF
	}
	n := copy(b, ws.buf[ws.i:])
	ws.i += int64(n)
	return n, nil
}

func (ws *bytesReadWriteSeeker) Write(b []byte) (int, error) {
	ws.buf = append(ws.buf, b...)
	return len(b), nil
}

// Copied from Go stdlib bytes.Reader
func (ws *bytesReadWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case 0:
		abs = offset
	case 1:
		abs = int64(ws.i) + offset
	case 2:
		abs = int64(len(ws.buf)) + offset
	default:
		return 0, errors.New("bytes.Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("bytes.Reader.Seek: negative position")
	}
	ws.i = abs
	return abs, nil
}
