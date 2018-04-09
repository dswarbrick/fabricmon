/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017-18 Daniel Swarbrick
 */

var fabrics = [
  ["2-switch topology", "samples/2-switch.json"],
  ["3-switch topology", "samples/3-switch.json"],
  ["fat tree topology", "samples/fat-tree.json"]
];

// nodeImageMap is an array of regex-URL pairs. The node description will be tested against each
// regex in the specified order, and the first match will determine the image used for the node.
// The last regex should be a catch-all pattern.
var nodeImageMap = [
  [/^gw(\d+[ab]-\d+|\d+)/, "img/router.svg"],
  [/^(n\d+[ab]-\d+|node\d+)/, "img/cpu.svg"],
  [/^(st\d+[ab]-\d+|storage\d+)/, "img/hdd.svg"],
  [/./, "img/default.svg"]
];
