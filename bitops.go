// Copyright 2017 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.
//
// Portions Copyright 2017 The Go Authors. All rights reserved.


// Low-level bit operations.
// TODO: Deprecate swapUint..() in favour of bits.ReverseBytes() from upcoming math/bits package.

package main

import (
	"encoding/binary"
	"unsafe"
)

var nativeEndian binary.ByteOrder

// Determine native endianness of system
func init() {
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		nativeEndian = binary.LittleEndian
	} else {
		nativeEndian = binary.BigEndian
	}
}

func bitLen(x uint) (n uint) {
	for ; x >= 0x8000; x >>= 16 {
		n += 16
	}
	if x >= 0x80 {
		x >>= 8
		n += 8
	}
	if x >= 0x8 {
		x >>= 4
		n += 4
	}
	if x >= 0x2 {
		x >>= 2
		n += 2
	}
	if x >= 0x1 {
		n++
	}
	return
}

func log2b(x uint) uint {
	return bitLen(x) - 1
}

// ntohs converts a uint16 from network byte order to host byte order
func ntohs(x uint16) uint16 {
	if nativeEndian == binary.LittleEndian {
		return swapUint16(x)
	}
	return x
}

// ntohll converts a uint32 from network byte order to host byte order
func ntohl(x uint32) uint32 {
	if nativeEndian == binary.LittleEndian {
		return swapUint32(x)
	}
	return x
}

// ntohll converts a uint64 from network byte order to host byte order
func ntohll(x uint64) uint64 {
	if nativeEndian == binary.LittleEndian {
		return swapUint64(x)
	}
	return x
}

func swapUint16(n uint16) uint16 {
	return (n&0x00ff)<<8 | (n&0xff00)>>8
}

func swapUint32(n uint32) uint32 {
	return (n&0x000000ff)<<24 | (n&0x0000ff00)<<8 |
		(n&0x00ff0000)>>8 | (n&0xff000000)>>24
}

func swapUint64(n uint64) uint64 {
	return ((n & 0x00000000000000ff) << 56) |
		((n & 0x000000000000ff00) << 40) |
		((n & 0x0000000000ff0000) << 24) |
		((n & 0x00000000ff000000) << 8) |
		((n & 0x000000ff00000000) >> 8) |
		((n & 0x0000ff0000000000) >> 24) |
		((n & 0x00ff000000000000) >> 40) |
		((n & 0xff00000000000000) >> 56)
}
