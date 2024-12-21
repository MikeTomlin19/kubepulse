import * as d3 from 'd3';
import { Node, Pod } from '../types/kubernetes';

export const createNodeElements = (nodeGroups: d3.Selection<SVGGElement, Node, SVGElement, unknown>) => {
  // Create node rectangles
  nodeGroups
    .append('rect')
    .attr('width', 200)
    .attr('height', 150)
    .attr('rx', 5)
    .attr('ry', 5)
    .attr('fill', '#f0f0f0')
    .attr('stroke', d => d.status === 'Ready' ? '#4CAF50' : '#f44336')
    .attr('stroke-width', 2);

  // Add node labels
  nodeGroups
    .append('text')
    .attr('x', 10)
    .attr('y', 20)
    .text(d => d.name)
    .attr('fill', '#333')
    .attr('font-weight', 'bold');

  // Add CPU usage bar
  nodeGroups
    .append('rect')
    .attr('x', 10)
    .attr('y', 30)
    .attr('width', 180)
    .attr('height', 10)
    .attr('fill', '#e0e0e0');

  nodeGroups
    .append('rect')
    .attr('x', 10)
    .attr('y', 30)
    .attr('width', d => (d.metrics.usage / d.metrics.capacity) * 180)
    .attr('height', 10)
    .attr('fill', '#2196F3');

  // Add memory usage bar
  nodeGroups
    .append('rect')
    .attr('x', 10)
    .attr('y', 45)
    .attr('width', 180)
    .attr('height', 10)
    .attr('fill', '#e0e0e0');

  nodeGroups
    .append('rect')
    .attr('x', 10)
    .attr('y', 45)
    .attr('width', d => (d.metrics.usage / d.metrics.capacity) * 180)
    .attr('height', 10)
    .attr('fill', '#4CAF50');
};

export const createPodElements = (podGroups: d3.Selection<SVGGElement, Pod, SVGGElement, Node>) => {
  // Create pod rectangles
  podGroups
    .append('rect')
    .attr('width', 40)
    .attr('height', 40)
    .attr('rx', 3)
    .attr('ry', 3)
    .attr('fill', '#fff')
    .attr('stroke', d => {
      switch (d.status) {
        case 'running': return '#4CAF50';
        case 'pending': return '#FFC107';
        default: return '#f44336';
      }
    })
    .attr('stroke-width', 2);

  // Add resource usage indicators
  podGroups
    .append('rect')
    .attr('x', 5)
    .attr('y', 30)
    .attr('width', 30)
    .attr('height', 5)
    .attr('fill', d => {
      const cpuUsage = d.metrics.CPU.usage / d.metrics.CPU.requests;
      return cpuUsage > 0.8 ? '#f44336' : '#4CAF50';
    });
};

export const updateNodePositions = (nodeGroups: d3.Selection<SVGGElement, Node, SVGElement, unknown>) => {
  nodeGroups
    .attr('transform', (d, i) => `translate(${i * 220 + 20}, 20)`);
};

export const updatePodPositions = (podGroups: d3.Selection<SVGGElement, Pod, SVGGElement, Node>) => {
  podGroups
    .attr('transform', (d, i) => `translate(${(i % 4) * 45 + 10}, ${Math.floor(i / 4) * 45 + 70})`);
};

export const createTooltip = () => {
  return d3.select('body')
    .append('div')
    .attr('class', 'tooltip')
    .style('position', 'absolute')
    .style('visibility', 'hidden')
    .style('background-color', 'white')
    .style('padding', '10px')
    .style('border-radius', '5px')
    .style('box-shadow', '0 2px 4px rgba(0,0,0,0.2)')
    .style('pointer-events', 'none');
};

export const showTooltip = (tooltip: d3.Selection<HTMLDivElement, unknown, HTMLElement, unknown>, content: string, event: MouseEvent) => {
  tooltip
    .style('visibility', 'visible')
    .html(content)
    .style('left', `${event.pageX + 10}px`)
    .style('top', `${event.pageY + 10}px`);
};

export const hideTooltip = (tooltip: d3.Selection<HTMLDivElement, unknown, HTMLElement, unknown>) => {
  tooltip.style('visibility', 'hidden');
};

