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
	"bytes"
	"unsafe"
)

// umadGetCADeviceList wraps the umad_get_ca_device_list() function. This function is the modern
// replacement for umad_get_cas_names(), and supports returning an arbitrary number of CAs. It is
// supported by rdma-core v26.0 and later.
func umadGetCADeviceList() []string {
	device_list := C.umad_get_ca_device_list()

	hcas := make([]string, 0)
	for node := device_list; node != nil; node = node.next {
		hcas = append(hcas, C.GoString(node.ca_name))
	}

	C.umad_free_ca_device_list(device_list)

	return hcas
}

// umadGetCANames returns a slice of CA names, as retrieved by libibumad. This function must be
// used when running FabricMon under ibsim, since the libumad2sim.so does not intercept Go's use
// of the openat() syscall.
//
// Deprecated: Use umadGetCADeviceList instead, which supports returning an arbitrary number of CAs.
func umadGetCANames() []string {
	var buf [C.UMAD_MAX_DEVICES][C.UMAD_CA_NAME_LEN]byte

	// Call umad_get_cas_names with pointer to first element in our buffer
	cas_found := C.umad_get_cas_names((*[C.UMAD_CA_NAME_LEN]C.char)(unsafe.Pointer(&buf[0])),
		C.UMAD_MAX_DEVICES)

	hcas := make([]string, 0, cas_found)
	for x := 0; x < int(cas_found); x++ {
		hcas = append(hcas, string(bytes.TrimRight(buf[x][:], "\x00")))
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
