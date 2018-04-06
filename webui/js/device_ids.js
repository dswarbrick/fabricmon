/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017-18 Daniel Swarbrick
 *
 * Device IDs from https://pci-ids.ucw.cz/
 */

// Note that InfiniBand vendor ID is not the same as PCI vendor ID!
deviceIds = {
  0x2c9: [
    "Mellanox",
    {
      0x1003: "MT27500 Family [ConnectX-3]",
      0x673c: "MT26428 [ConnectX VPI PCIe 2.0 5GT/s - IB QDR / 10GigE]",
      0xc738: "MT51136 SwitchX-2, 40GbE switch",
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
