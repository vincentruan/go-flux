package fluxmap

import (
	"errors"
	"io"
)

var (
	//ErrBadTag blah blah blah
	ErrBadTag = errors.New("Bad tag.")
)

//Reader must implement io.Reader, io.ByteReader, and be able to unread a byte
//*bytes.Buffer satisfies this interface, but you may use your own
type Reader interface {
	io.Reader
	io.ByteReader
	//UnreadByte should unread the last read byte, and return an error on failure
	UnreadByte() error
}

func readBool(r Reader) (b bool, err error) {
	c, err := r.ReadByte()
	if err != nil {
		return
	}
	switch c {
	case mtrue:
		b = true
	case mfalse:
		b = false
	default:
		err = ErrBadTag
	}
	return
}

func readInt(r Reader) (i int64, err error) {
	var bs [8]byte
	var n byte
	c, err := r.ReadByte()
	if err != nil {
		return
	}
	//positive fixint
	if (c & 0x80) == 0 {
		i = int64(c & 0x7f)
		return
	}
	//negative fixint
	if (c & 0xe0) == 0 {
		i = int64(c & 0x1f)
		return
	}

	switch c {
	case mint8:
		n, err = r.ReadByte()
		if err != nil {
			return
		}
		i = int64(n)
		return
	case mint16:
		_, err = r.Read(bs[:2])
		if err != nil {
			return
		}
		i = int64(uint16(bs[1]) | uint16(bs[0])<<8)
		return
	case mint32:
		_, err = r.Read(bs[:4])
		if err != nil {
			return
		}
		i = int64(uint32(bs[3]) |
			uint32(bs[2]<<8) |
			uint32(bs[1])<<16 |
			uint32(bs[0])<<24)
		return
	case mint64:
		_, err = r.Read(bs[:8])
		if err != nil {
			return
		}
		i = int64(uint64(bs[7]) |
			uint64(bs[6])<<8 |
			uint64(bs[5])<<16 |
			uint64(bs[4])<<24 |
			uint64(bs[3])<<32 |
			uint64(bs[2])<<40 |
			uint64(bs[1])<<48 |
			uint64(bs[0])<<56)
	default:
		err = ErrBadTag
		return
	}
	return
}

func readUint(r Reader) (i uint64, err error) {
	var bs [8]byte
	var c byte
	c, err = r.ReadByte()
	if err != nil {
		return
	}
	//positive fixint
	if (c & 0x80) == 0 {
		i = uint64(c & 0x7f)
		return
	}

	switch c {
	case muint8:
		_, err = r.Read(bs[:1])
		if err != nil {
			return
		}
		i = uint64(bs[0])
		return

	case muint16:
		_, err = r.Read(bs[:2])
		if err != nil {
			return
		}
		i = uint64(uint16(bs[1]) | uint16(bs[0])<<8)
		return

	case muint32:
		_, err = r.Read(bs[:4])
		if err != nil {
			return
		}
		i = uint64(uint32(bs[3]) |
			uint32(bs[2])<<8 |
			uint32(bs[1])<<16 |
			uint32(bs[0])<<24)
		return

	case muint64:
		_, err = r.Read(bs[:8])
		if err != nil {
			return
		}
		i = uint64(uint64(bs[7]) |
			uint64(bs[6])<<8 |
			uint64(bs[5])<<16 |
			uint64(bs[4])<<24 |
			uint64(bs[3])<<32 |
			uint64(bs[2])<<40 |
			uint64(bs[1])<<48 |
			uint64(bs[0])<<56)
		return

	default:
		err = ErrBadTag
		return
	}

}

func readString(r Reader) (s string, err error) {
	var bs [31]byte //for short strings (fixstr) - we can avoid allocating
	var ns [4]byte  //for length
	var bsl []byte  //slice for dynamic strings
	var c byte      //leading byte
	var n uint32    //len

	c, err = r.ReadByte()
	if err != nil {
		return
	}

	//shortcut fixstr case
	//mask 11100000 should be 10100000
	if c&0x70 == 0x50 {
		//mask 00011111
		n = uint32(c & 0x1f)
		if n > 31 {
			panic("Impossible")
		}
		_, err = r.Read(bs[:n])
		if err != nil {
			return
		}
		s = string(bs[:n])
		return
	}

	//determine length
	switch c {
	case mstr8:
		c, err = r.ReadByte()
		if err != nil {
			return
		}
		n = uint32(c)

	case mstr16:
		_, err = r.Read(ns[:2])
		if err != nil {
			return
		}
		n = uint32(uint16(ns[1]) | uint16(ns[0])<<8)

	case mstr32:
		_, err = r.Read(ns[:4])
		if err != nil {
			return
		}
		n = uint32(uint32(ns[3]) | uint32(ns[2])<<8 | uint32(ns[1])<<16 | uint32(ns[0])<<24)

	default:
		err = ErrBadTag
		return
	}
	//make slice; read
	bsl = make([]byte, n)
	_, err = r.Read(bsl)
	if err != nil {
		return
	}
	s = string(bsl)
	return
}

//read binary into p
func readBin(r Reader, p []byte) (err error) {
	var c byte     //leading byte
	var n uint32   //length
	var ns [4]byte //for length bytes

	c, err = r.ReadByte()
	if err != nil {
		return
	}

	//find n
	switch c {
	case mbin8:
		c, err = r.ReadByte()
		if err != nil {
			return
		}
		n = uint32(c)

	case mbin16:
		_, err = r.Read(ns[:2])
		if err != nil {
			return
		}
		n = uint32(uint16(ns[1]) | uint16(ns[0])<<8)

	case mbin32:
		_, err = r.Read(ns[:4])
		if err != nil {
			return
		}
		n = uint32(uint32(ns[3]) | uint32(ns[2])<<8 | uint32(ns[1])<<16 | uint32(ns[0])<<24)

	default:
		err = ErrBadTag
		return

	}
	//read into p; return
	if cap(p) < int(n) {
		//allocate if p is not long enough
		p = make([]byte, n)
	} else {
		p = p[:n]
	}
	_, err = r.Read(p[:n])
	return
}

func readExt(r Reader, dat []byte) (etype int8, err error) {
	var c byte //leading byte

	c, err = r.ReadByte()
	if err != nil {
		return
	}

	switch c {
	case mfixext1:
		etype, err = readfixExt(r, dat, 1)
		return
	case mfixext2:
		etype, err = readfixExt(r, dat, 2)
		return
	case mfixext4:
		etype, err = readfixExt(r, dat, 4)
		return
	case mfixext8:
		etype, err = readfixExt(r, dat, 8)
		return
	case mfixext16:
		etype, err = readfixExt(r, dat, 16)
		return
	}

	var n uint32   //length
	var ns [4]byte //length bytes

	//read length
	switch c {
	case mext8:
		ns[0], err = r.ReadByte()
		if err != nil {
			return
		}
		n = uint32(ns[0])

	case mext16:
		_, err = r.Read(ns[:2])
		if err != nil {
			return

		}
		n = uint32(uint16(ns[1]) | uint16(ns[0])<<8)

	case mext32:
		_, err = r.Read(ns[:4])
		if err != nil {
			return
		}
		n = uint32(uint32(ns[3]) | uint32(ns[2])<<8 | uint32(ns[1])<<16 | uint32(ns[0])<<24)

	default:
		err = ErrBadTag
		return
	}

	//read extension type
	c, err = r.ReadByte()
	if err != nil {
		return
	}
	etype = int8(c)

	if cap(dat) < int(n) {
		dat = make([]byte, n)
	} else {
		dat = dat[:n]
	}
	_, err = r.Read(dat)

	return
}

func readfixExt(r Reader, dat []byte, size uint8) (etype int8, err error) {
	if size > 16 {
		panic("Impossible.")
	}

	var c byte
	c, err = r.ReadByte()
	if err != nil {
		return
	}
	etype = int8(c)

	if cap(dat) >= int(size) {
		dat = dat[:size]
		_, err = r.Read(dat)
		return
	}

	var bs [16]byte //max size
	_, err = r.Read(bs[:size])
	if err != nil {
		return
	}
	dat = bs[:size]
	return

}

//returns the length of the map
func readMapHeader(r Reader) (n uint32, err error) {
	var c byte
	c, err = r.ReadByte()
	if err != nil {
		return
	}

	//fixmap case
	//if c & 11110000 == 10000000, b/c fixmap is 1000XXXX
	if (c & 0xf0) == 0x80 {
		//return c & 00001111
		n = uint32(c & 0xf)
		return
	}

	var ns [4]byte
	switch c {
	case mmap16:
		_, err = r.Read(ns[:2])
		n = uint32(uint16(ns[1]) | uint16(ns[0])<<8)

	case mmap32:
		_, err = r.Read(ns[:4])
		n = uint32(uint32(ns[3]) | uint32(ns[2])<<8 | uint32(ns[1])<<16 | uint32(ns[0])<<24)

	default:
		err = ErrBadTag
		return

	}
	return
}

//does nothing unless leading byte is not mnil - then error
func readNil(r Reader) (err error) {
	c, err := r.ReadByte()
	if err != nil {
		return
	}
	if c != mnil {
		err = ErrBadTag
	}
	return
}
