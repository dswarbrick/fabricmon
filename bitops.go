// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Low-level bit operations.

package main

import (
	"encoding/binary"
	"math/bits"
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

func log2b(x uint) uint {
	// FIXME: This can wrap if bits.Len() returns zero
	return uint(bits.Len(x) - 1)
}

// ntohs converts a uint16 from network byte order to host byte order
func ntohs(x uint16) uint16 {
	if nativeEndian == binary.LittleEndian {
		return bits.ReverseBytes16(x)
	}
	return x
}

// ntohll converts a uint32 from network byte order to host byte order
func ntohl(x uint32) uint32 {
	if nativeEndian == binary.LittleEndian {
		return bits.ReverseBytes32(x)
	}
	return x
}

// ntohll converts a uint64 from network byte order to host byte order
func ntohll(x uint64) uint64 {
	if nativeEndian == binary.LittleEndian {
		return bits.ReverseBytes64(x)
	}
	return x
}
