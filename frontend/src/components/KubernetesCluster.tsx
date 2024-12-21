import React, { useRef, useEffect } from 'react';
import * as d3 from 'd3';
import { ClusterData, Node, Pod } from '../types/kubernetes';
import {
  createNodeElements,
  createPodElements,
  updateNodePositions,
  updatePodPositions,
  createTooltip,
  showTooltip,
  hideTooltip
} from '../utils/d3Utils';

interface KubernetesClusterProps {
  data: ClusterData;
}

export const KubernetesCluster: React.FC<KubernetesClusterProps> = ({ data }) => {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    if (!svgRef.current || !data) return;

    const svg = d3.select(svgRef.current);
    const tooltip = createTooltip();

    // Clear previous content
    svg.selectAll('*').remove();

    // Create node groups
    const nodeGroups = svg.selectAll<SVGGElement, Node>('g.node')
      .data(data.nodes, d => d.id)
      .enter()
      .append('g')
      .attr('class', 'node');

    // Create node elements
    createNodeElements(nodeGroups);

    // Create pod groups
    const podGroups = nodeGroups.selectAll<SVGGElement, Pod>('g.pod')
      .data(d => d.pods, d => d.id)
      .enter()
      .append('g')
      .attr('class', 'pod');

    // Create pod elements
    createPodElements(podGroups);

    // Update node and pod positions
    updateNodePositions(nodeGroups);
    updatePodPositions(podGroups);

    // Add tooltips
    nodeGroups
      .on('mouseover', (event: MouseEvent, d: Node) => {
        const content = `
          <strong>${d.name}</strong><br>
          Status: ${d.status}<br>
          CPU: ${d.metrics.usage}/${d.metrics.capacity} millicores<br>
          Memory: ${d.metrics.usage}/${d.metrics.capacity} bytes
        `;
        showTooltip(tooltip, content, event);
      })
      .on('mouseout', () => hideTooltip(tooltip));

    podGroups
      .on('mouseover', (event: MouseEvent, d: Pod) => {
        const content = `
          <strong>${d.name}</strong><br>
          Namespace: ${d.namespace}<br>
          Status: ${d.status}<br>
          CPU: ${d.metrics.CPU.usage} millicores<br>
          Memory: ${d.metrics.Memory.usage} bytes
        `;
        showTooltip(tooltip, content, event);
      })
      .on('mouseout', () => hideTooltip(tooltip));

    // Animate pod creation and termination
    podGroups
      .attr('opacity', 0)
      .transition()
      .duration(500)
      .attr('opacity', 1);

    podGroups.exit()
      .transition()
      .duration(500)
      .attr('opacity', 0)
      .remove();

  }, [data]);

  return (
    <svg ref={svgRef} width="100%" height="600" />
  );
};

