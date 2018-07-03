# FabricMon

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

## Building FabricMon

To build FabricMon, you will require the following development libraries
(Debian package names shown):

* libibmad-dev
* libibumad-dev
* libibnetdisc-dev
* libopensm-dev

The corresponding runtime libraries will be required on the target system
unless you build the FabricMon binary with static linking.

## InfiniBand Counters

InfiniBand port counters do not automatically wrap when they reach their
maximum possible value, and instead latch with all bits set to one. In the case
of the 64-bit extended counters, this is likely to take a very long time, but
some of the error counters are 16, 8 or even 4 bits wide.

Note that counters that represent data (e.g. PortXmitData and PortRcvData) are
divided by four (lanes). See https://community.mellanox.com/docs/DOC-2572 for
more information.

### Error Counters

The following counters are *less than* 32 bits wide:

| Counter                      | Bits |
| ---------------------------- | ---- |
| SymbolErrorCounter           | 16   |
| LinkErrorRecoveryCounter     | 8    |
| LinkDownedCounter            | 8    |
| PortRcvErrors                | 16   |
| PortRcvRemotePhysicalErrors  | 16   |
| PortRcvSwitchRelayErrors     | 16   |
| PortXmitDiscards             | 16   |
| PortXmitContraintErrors      | 8    |
| PortRcvConstraintErrors      | 8    |
| LocalLinkIntegrityErrors     | 4    |
| ExcessiveBufferOverrunErrors | 4    |
| QP1Dropped                   | 16   |
| VL15Dropped                  | 16   |

(cf. Table 247, InfiniBand Architecture Release 1.3, Volume I)

## Testing

Start ibsim:

```
$ ibsim -s ibsim.net
```

Run fabricmon with an LD_PRELOAD, so that it will connect to the simulated
fabric:

```
$ LD_PRELOAD=/usr/lib/umad2sim/libumad2sim.so go run *.go
```

## InfluxDB Data Model

Counters are written to InfluxDB in a simple key -> value style. Tags include the following:

 * host - host from whence the counters were scraped (i.e., FabricMon host)
 * hca - InfiniBand HCA connected to the fabric
 * src_port - HCA port from which the fabric discovery was performed
 * guid - InfiniBand switch GUID
 * port - InfiniBand switch port number
 * counter - InfiniBand counter name

The value is an integer field. Note that InfluxDB < 1.6 does not support uint64 values, whereas
several of the InfiniBand counters utilize the full 64 bits. Counter values will be truncated to
63 bits, so that they will fit in a signed int64 field.

### Example InfluxDB Measurement

```
time                counter                     guid             hca    host    port src_port value
----                -------                     ----             ---    ----    ---- -------- -----
1521657387000000000 VL15Dropped                 003048ffff5812fc ibsim0 hal9000 1    0        2160
1521657387000000000 SymbolErrorCounter          003048ffff5812fc ibsim0 hal9000 1    0        0
1521657387000000000 PortXmitPkts                003048ffff5812fc ibsim0 hal9000 1    0        30
1521657387000000000 PortXmitDiscards            003048ffff5812fc ibsim0 hal9000 1    0        30
1521657387000000000 PortXmitData                003048ffff5812fc ibsim0 hal9000 1    0        2160
1521657387000000000 PortXmitConstraintErrors    003048ffff5812fc ibsim0 hal9000 1    0        30
1521657387000000000 PortUnicastXmitPkts         003048ffff5812fc ibsim0 hal9000 1    0        0
1521657387000000000 PortUnicastRcvPkts          003048ffff5812fc ibsim0 hal9000 1    0        0
1521657387000000000 PortRcvSwitchRelayErrors    003048ffff5812fc ibsim0 hal9000 1    0        30
1521657387000000000 PortRcvRemotePhysicalErrors 003048ffff5812fc ibsim0 hal9000 1    0        2160
1521657387000000000 VL15Dropped                 003048ffff5812fc ibsim0 hal9000 2    0        0
1521657387000000000 SymbolErrorCounter          003048ffff5812fc ibsim0 hal9000 2    0        0
1521657387000000000 PortXmitPkts                003048ffff5812fc ibsim0 hal9000 2    0        30
1521657387000000000 PortXmitDiscards            003048ffff5812fc ibsim0 hal9000 2    0        0
1521657387000000000 PortXmitData                003048ffff5812fc ibsim0 hal9000 2    0        2160
1521657387000000000 PortXmitConstraintErrors    003048ffff5812fc ibsim0 hal9000 2    0        2160
1521657387000000000 PortUnicastXmitPkts         003048ffff5812fc ibsim0 hal9000 2    0        0
1521657387000000000 PortUnicastRcvPkts          003048ffff5812fc ibsim0 hal9000 2    0        0
1521657387000000000 PortRcvSwitchRelayErrors    003048ffff5812fc ibsim0 hal9000 2    0        0
1521657387000000000 PortRcvRemotePhysicalErrors 003048ffff5812fc ibsim0 hal9000 2    0        0
```

## Future Plans

* Subscribe to SM traps 128 (link state change) and 144 (port capabilities
  change), to avoid performing full sweep upon each poll.
