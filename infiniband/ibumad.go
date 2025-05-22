// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Functions analogous to libibumad.
// Note: When running in an ibsim environment, the libumad2sim.so LD_PRELOAD hijacks libc syscall
// wrappers such as scandir(3), which libibumad uses to enumerate HCAs found in sysfs. Other libc
// functions like ioctl(2) and basic file IO functions (e.g., open(2), read(2) etc.) are also
// hijacked to intercept operations on /dev/infiniband/* and /sys/class/infiniband/* entries.
//
// Go's os.ReadDir() function results in a function call chain of:
//   os.Open -> os.OpenFile -> syscall.Open -> syscall.openat -> syscall.Syscall6(SYS_OPENAT, ...)
// The openat() call is not intercepted by the libumad2sim.so LD_PRELOAD.

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #cgo LDFLAGS: -libumad
// #include <umad.h>
import "C"

import (
	"strings"
	"unsafe"
)

// UmadGetCANames returns a slice of CA names, as retrieved by libibumad. This function must be
// used when running FabricMon under ibsim, since the libumad2sim.so does not intercept Go's use
// of the openat() syscall.
func umadGetCANames() []string {
	var (
		buf  [C.UMAD_MAX_DEVICES][C.UMAD_CA_NAME_LEN]byte
		hcas = make([]string, 0, C.UMAD_MAX_DEVICES)
	)

	// Call umad_get_cas_names with pointer to first element in our buffer
	numHCAs := C.umad_get_cas_names((*[C.UMAD_CA_NAME_LEN]C.char)(unsafe.Pointer(&buf[0])), C.UMAD_MAX_DEVICES)

	for x := 0; x < int(numHCAs); x++ {
		hcas = append(hcas, strings.TrimRight(string(buf[x][:]), "\x00"))
	}

	return hcas
}

// UmadDone wraps the umad_done() function. Under the hood, umad_done() does nothing.
//
// Deprecated.
func UmadDone() int {
	return int(C.umad_done())
}

// UmadInit wraps the umad_init() function. Since rdma-core v26.0, umad_init() does nothing.
//
// Deprecated.
func UmadInit() int {
	return int(C.umad_init())
}
