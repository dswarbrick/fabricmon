/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * Low-level bit operations
 * Portions Copyright 2017 The Go Authors. All rights reserved.
 *
 * TODO: Deprecate swapUint..() in favour of bits.ReverseBytes() from upcoming math/bits package
 */
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

// ntohll converts a uint64 from network byte order to host byte order
func ntohll(x uint64) uint64 {
	if nativeEndian == binary.LittleEndian {
		return swapUint64(x)
	}
	return x
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
