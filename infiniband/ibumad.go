// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Functions analogous to libibumad.
// Note: When running in an ibsim environment, the libumad2sim.so LD_PRELOAD hijacks libc syscall
// wrappers such as scandir(3), which libibumad uses to enumerate HCAs found in sysfs. Other libc
// functions like ioctl(2) and basic file IO functions (e.g., open(2), read(2) etc.) are also
// hijacked to intercept operations on /dev/infiniband/* and /sys/class/infiniband/* entries.
//
// Go's ioutil.ReadDir() function results in a function call chain of:
//   os.Open -> os.OpenFile -> syscall.Open -> syscall.openat -> syscall.Syscall6(SYS_OPENAT, ...)
// The openat() call is not intercepted by the libumad2sim.so LD_PRELOAD.

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #cgo LDFLAGS: -libmad -libumad
// #include <umad.h>
import "C"

import (
	"io/ioutil"
	"strings"
	"unsafe"
)

const (
	SYS_INFINIBAND = "/sys/class/infiniband"
)

// UmadInit simply wraps the libibumad umad_init() function.
func UmadInit() int {
	return int(C.umad_init())
}

// UmadDone simply wraps the libibumad umad_done() function.
func UmadDone() {
	// NOTE: ibsim indicates that FabricMon is not "disconnecting" when it exits - resource leak?
	C.umad_done()
}

// getCANames is the functional equivalent of umad_get_cas_names()
func getCANames() ([]string, error) {
	files, err := ioutil.ReadDir(SYS_INFINIBAND)
	if err != nil {
		return nil, err
	}

	caNames := []string{}
	for _, file := range files {
		caNames = append(caNames, file.Name())
	}

	return caNames, nil
}

// UmadGetCANames returns a slice of CA names, as retrieved by libibumad. This function must be
// used when running FabricMon under ibsim, since the libumad2sim.so does not intercept Go's use
// of the openat() syscall.
func umadGetCANames() []string {
	var (
		buf  [C.UMAD_CA_NAME_LEN][C.UMAD_MAX_DEVICES]byte
		hcas = make([]string, 0, C.UMAD_MAX_DEVICES)
	)

	// Call umad_get_cas_names with pointer to first element in our buffer
	numHCAs := C.umad_get_cas_names((*[C.UMAD_CA_NAME_LEN]C.char)(unsafe.Pointer(&buf[0])), C.UMAD_MAX_DEVICES)

	for x := 0; x < int(numHCAs); x++ {
		hcas = append(hcas, strings.TrimRight(string(buf[x][:]), "\x00"))
	}

	return hcas
}
