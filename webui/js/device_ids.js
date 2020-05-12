/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017-20 Daniel Swarbrick
 *
 * Device IDs from https://pci-ids.ucw.cz/
 */

// Note that InfiniBand vendor ID is not the same as PCI vendor ID!
deviceIds = {
  0x2c9: [
    "Mellanox",
    {
      0x1003: "MT27500 Family [ConnectX-3]",
      0x1005: "MT27510 Family",
      0x1007: "MT27520 ConnectX-3 Pro Family",
      0x1009: "MT27530 Family",
      0x100b: "MT27540 Family",
      0x100d: "MT27550 Family",
      0x100f: "MT27560 Family",
      0x1011: "MT27600 [Connect-IB]",
      0x1013: "MT27620 [ConnectX-4]",
      0x1015: "MT27630 Family [ConnectX-4LX]",
      0x1017: "MT27800 Family [ConnectX-5]",
      0x1019: "MT28800 Family [ConnectX-5, Ex]",
      0x634a: "MT25418 [ConnectX VPI PCIe 2.0 2.5GT/s - IB DDR / 10GigE]",
      0x6368: "MT25448 [ConnectX EN 10GigE, PCIe 2.0 2.5GT/s]",
      0x6372: "MT25458 [ConnectX EN 10GigE 10GBaseT, PCIe 2.0 2.5GT/s]",
      0x6430: "MT25408 [ConnectX VPI - IB SDR / 10GigE]",
      0x6732: "MT26418 [ConnectX VPI PCIe 2.0 5GT/s - IB DDR / 10GigE]",
      0x673c: "MT26428 [ConnectX VPI PCIe 2.0 5GT/s - IB QDR / 10GigE]",
      0x6746: "MT26438 [ConnectX-2 VPI w/ Virtualization+]",
      0x6750: "MT26448 [ConnectX EN 10GigE, PCIe 2.0 5GT/s]",
      0x675a: "MT26458 [ConnectX EN 10GigE 10GBaseT, PCIe Gen2 5GT/s]",
      0x6764: "MT26468 [Mountain top]",
      0x676e: "MT26478 [ConnectX EN 10GigE, PCIe 2.0 5GT/s]",
      0xbd34: "IS4 IB SDR",
      0xbd35: "IS4 IB DDR",
      0xbd36: "IS4 IB QDR",
      0xc738: "MT51136 SwitchX-2, 40GbE switch",
      0xcb20: "Switch-IB",
      0xcb84: "Spectrum",
      0xcf08: "Switch-IB 2",
      0xa2d2: "MT416842 Family BlueField ConnectX-5"
    }
  ]
}

function lookupDevice(vendorId, deviceId) {
  var device, vendor = deviceIds[vendorId];

  if (vendor != undefined) {
    device = vendor[1][deviceId];

    if (device != undefined) {
      return vendor[0] + " " + device;
    } else {
      console.log("Unknown vendor:device " + vendorId.toString(16) + ":" + deviceId.toString(16));
      return "Unknown " + vendor[0];
    }
  }

  console.log("Unknown vendor:device " + vendorId.toString(16) + ":" + deviceId.toString(16));
  return "Unknown";
}
