/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017-18 Daniel Swarbrick
 */

var nodeTypes = {1: "HCA", 2: "Switch", 3: "Router"},
  simulation, viewPort;

function dragstarted(d) {
  if (!d3.event.active) simulation.alphaTarget(0.3).restart();
  d.fx = d.x;
  d.fy = d.y;
}

function dragged(d) {
  d.fx = d3.event.x;
  d.fy = d3.event.y;
}

function dragended(d) {
  if (!d3.event.active) simulation.alphaTarget(0);
  d.fx = null;
  d.fy = null;
}

function clearSelection() {
  viewPort.select(".links").selectAll("line").style("opacity", 1);
  viewPort.selectAll(".node").style("opacity", 1);
  d3.selectAll("table.node-info td").text("-");
}

function handleNodeClick(d) {
  var links = simulation.force("link").links(),
    connectedNodes = [d.index];

  console.log("Node clicked: ", d);
  d3.select("#sel_node_type").text(nodeTypes[d.nodetype]);
  d3.select("#sel_node_guid").text(d.id);
  d3.select("#sel_node_desc").text(d.desc);

  if (d.vendor_id)
    d3.select("#sel_node_vendor_id").text("0x" + d.vendor_id.toString(16));

  if (d.device_id)
    d3.select("#sel_node_device_id").text("0x" + d.device_id.toString(16));

  if (d.vendor_id && d.device_id)
    d3.select("#sel_node_model").text(lookupDevice(d.vendor_id, d.device_id));

  // Build array of nodes that are linked to this node
  links.forEach(function (l) {
    if (l.source.index == d.index) connectedNodes.push(l.target.index);
    if (l.target.index == d.index) connectedNodes.push(l.source.index);
  });

  // Adjust opacity of links depending on whether it is connected to this node
  viewPort.select(".links").selectAll("line").style("opacity", function(o) {
    if (o.source.index == d.index || o.target.index == d.index)
      return 1;
    else
      return 0.1;
  });

  // Adjust opacity of nodes dependong on whether they are connected to this node
  viewPort.selectAll(".node").style("opacity", function(o) {
    if (connectedNodes.indexOf(o.index) != -1)
      return 1;
    else
      return 0.1;
  });

  d3.event.stopPropagation();
}

function changeFabric() {
  // Clear existing fabric
  viewPort.selectAll("g").remove();

  var jsonUrl = this.value;

  if (jsonUrl === "")
    return;
  else
    console.log("Loading " + jsonUrl);

  // Container group which will be used for zooming
  var g = viewPort.append("g");

  // Attach zoom event handler to <svg> element
  viewPort.call(d3.zoom()
    .scaleExtent([0.5, 4])
    .on("zoom", function() { g.attr("transform", d3.event.transform); }));

  var bbox = viewPort.node().getBBox();

  simulation = d3.forceSimulation()
    .force("link", d3.forceLink()
      .distance(120)
      .id(function(d) { return d.id; }))
    .force("charge", d3.forceManyBody()
      .strength(-500))
    .force("center", d3.forceCenter(bbox.width / 2, bbox.height / 2));

  d3.json(jsonUrl, function(error, graph) {
    if (error) throw error;

    var link = g.append("g")
      .attr("class", "links")
      .selectAll("line")
      .data(graph.links)
      .enter().append("line")
        .attr("stroke-width", function(d) {
          if (d.value)
            return Math.sqrt(d.value);
          else
            return 1;
        });

    var node = g.selectAll(".node")
      .data(graph.nodes)
      .enter().append("g")
        .attr("class", "node")
        .call(d3.drag()
          .on("start", dragstarted)
          .on("drag", dragged)
          .on("end", dragended))
        .on("click", handleNodeClick);

    node.append("image")
      .attr("xlink:href", function(d) {
        if (d.nodetype == 2) {
          return "img/switch.svg";
        } else if (d.nodetype == 3) {
          return "img/router.svg";
        } else {
          for (var i = 0; i < nodeImageMap.length; i++) {
            if (nodeImageMap[i][0].test(d.desc))
              return nodeImageMap[i][1];
          }
        }
      })
      .attr("x", -8)
      .attr("y", -8)
      .attr("width", 16)
      .attr("height", 16);

    node.append("text")
      .attr("dx", 12)
      .attr("dy", ".35em")
      .text(function(d) { return d.desc.replace(/ HCA-\d+.*/, ""); });

    node.append("title")
      .text(function(d) { return nodeTypes[d.nodetype] || "Unknown node type"; });

    simulation
      .nodes(graph.nodes)
      .on("tick", ticked);

    simulation.force("link")
      .links(graph.links);

    function ticked() {
      link
        .attr("x1", function(d) { return d.source.x; })
        .attr("y1", function(d) { return d.source.y; })
        .attr("x2", function(d) { return d.target.x; })
        .attr("y2", function(d) { return d.target.y; });

      node.attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });
    }
  });
}

document.onreadystatechange = () => {
  if (document.readyState === 'complete') {
    // Populate fabric-select list
    d3.select("#fabric_select")
      .on("change", changeFabric)
      .selectAll()
      .data(fabrics).enter()
        .append("option")
          .attr("value", function(d) { return d[1]; })
          .text(function(d) { return d[0]; });

    viewPort = d3.select("svg")
      .on("click", clearSelection);
  }
};
