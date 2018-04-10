// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Low-level bit operations.

package infiniband

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

// MaxPow2Divisor calculates the highest power of two divisor shared by two non-negative integers.
// This is useful for finding the highest bit enum shared by two values. If x and y do not share
// any common bits, the result is zero.
func MaxPow2Divisor(x, y uint) uint {
	return 1 << uint(bits.Len(x&y)-1)
}

// htons converts a uint16 from host byte order to network byte order
func Htons(x uint16) uint16 {
	if nativeEndian != binary.BigEndian {
		return bits.ReverseBytes16(x)
	}
	return x
}

// ntohs converts a uint16 from network byte order to host byte order
func Ntohs(x uint16) uint16 {
	if nativeEndian != binary.BigEndian {
		return bits.ReverseBytes16(x)
	}
	return x
}

// ntohll converts a uint32 from network byte order to host byte order
func Ntohl(x uint32) uint32 {
	if nativeEndian != binary.BigEndian {
		return bits.ReverseBytes32(x)
	}
	return x
}

// ntohll converts a uint64 from network byte order to host byte order
func Ntohll(x uint64) uint64 {
	if nativeEndian != binary.BigEndian {
		return bits.ReverseBytes64(x)
	}
	return x
}
