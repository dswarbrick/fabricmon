/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 */

var nodeTypes = {1: "HCA", 2: "Switch", 3: "Router"},
  simulation;

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

function handleNodeClick(d) {
  var links = simulation.force("link").links(),
    connectedNodes = [d.index];

  console.log("Node clicked: ", d);
  document.getElementById("sel_node_guid").textContent = d.id;
  document.getElementById("sel_node_desc").textContent = d.desc;

  // Build array of nodes that are linked to this node
  links.forEach(function (l) {
    if (l.source.index == d.index) connectedNodes.push(l.target.index);
    if (l.target.index == d.index) connectedNodes.push(l.source.index);
  });

  // Adjust opacity of links depending on whether it is connected to this node
  d3.select("svg").selectAll(".links").selectAll("line").style("opacity", function(o) {
    if (o.source.index == d.index || o.target.index == d.index)
      return 1;
    else
      return 0.1;
  });

  // Adjust opacity of nodes dependong on whether they are connected to this node
  d3.select("svg").selectAll(".node").style("opacity", function(o) {
    if (connectedNodes.indexOf(o.index) != -1)
      return 1;
    else
      return 0.1;
  });
}

function changeFabric(event) {
  var svg = d3.select("svg");

  // Clear existing fabric
  svg.selectAll("g").remove();

  // Container group which will be used for zooming
  var g = svg.append("g");

  // Attach zoom event handler to <svg> element
  svg.call(d3.zoom()
    .scaleExtent([0.5, 3])
    .on("zoom", function() { g.attr("transform", d3.event.transform); }));

  simulation = d3.forceSimulation()
    .force("link", d3.forceLink()
      .distance(120)
      .id(function(d) { return d.id; }))
    .force("charge", d3.forceManyBody()
      .strength(-80))
    .force("center", d3.forceCenter(bbox.width / 2, bbox.height / 2));

  d3.json(event.target.value, function(error, graph) {
    if (error) throw error;

    var link = g.append("g")
      .attr("class", "links")
      .selectAll("line")
      .data(graph.links)
      .enter().append("line")
        .attr("stroke-width", function(d) { if (d.value) return Math.sqrt(d.value); else return 1; });

    var node = g.selectAll(".node")
      .data(graph.nodes)
      .enter().append("g")
        .attr("class", "node")
        .call(d3.drag()
          .on("start", dragstarted)
          .on("drag", dragged)
          .on("end", dragended));

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

    node.on("click", handleNodeClick);

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

var svg = d3.select("svg"),
  bbox = svg.node().getBBox();

var sel = document.getElementById("fabric_select");

// Populate fabric-select list
for (var x = 0; x < fabrics.length; x++) {
  var option = document.createElement("option");
  option.text = fabrics[x][0];
  option.value = fabrics[x][1];
  sel.add(option);
}

sel.addEventListener("change", changeFabric);
