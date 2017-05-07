/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 */

// URLSearchParams is not supported by MSIE / Edge!
var params = new URLSearchParams(window.location.search);
var fabric = params.get("fabric");

if (fabric === null) {
  window.alert("Please specify a fabric in URL query string.");
}

var nodeTypes = {1: "HCA", 2: "Switch", 3: "Router"}

var svg = d3.select("svg"),
  bbox = svg.node().getBBox();

// Container group which will be used for zooming
var g = svg.append("g");

// Attach zoom event handler to <svg> element
svg.call(d3.zoom()
  .scaleExtent([0.5, 3])
  .on("zoom", function() { g.attr("transform", d3.event.transform); }));

var simulation = d3.forceSimulation()
  .force("link", d3.forceLink()
    .distance(120)
    .id(function(d) { return d.id; }))
  .force("charge", d3.forceManyBody()
    .strength(-80))
  .force("center", d3.forceCenter(bbox.width / 2, bbox.height / 2));

d3.json(fabric + ".json", function(error, graph) {
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
