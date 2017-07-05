/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * Functions analogous to libibumad
 *
 * Note: Fabricmon cannot currently be run in an ibsim environment. The libumad2sim.so LD_PRELOAD
 * hijacks libc functions such as scandir(3), which libibumad uses to enumerate HCAs found in sysfs.
 * Other libc functions like ioctl(2) and basic file IO functions (e.g., open(2), read(2) etc.) are
 * also hijacked to intercept operations on /dev/infiniband/* and /sys/class/infiniband/* entries.
 */
package main

import "io/ioutil"

const SYS_INFINIBAND = "/sys/class/infiniband"

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
