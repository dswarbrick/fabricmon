FabricMon
=========

FabricMon is an InfiniBand fabric monitoring daemon written in Go. It uses cgo
to call low-level functions in libibmad, libibumad, and libibnetdiscover.

InfiniBand switch modules for blade chassis are often unmanaged, with no simple
way to query their port counters. FabricMon solves this by querying the subnet
manager (SM), using management datagrams (MAD). The topology of the fabric is
mapped using libibnetdiscover and the counters of any active switch ports
are queried.

The fabric topology is also offered as a .JSON file, which is parsed by
FabricMon's web interface, based on the d3.js graph library, and displayed as
an SVG force graph.

This project is a work in progress, in the early stages of development.
