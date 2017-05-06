/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * Functions analogous to libibumad
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
