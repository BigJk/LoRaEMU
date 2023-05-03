import * as d3 from 'd3';
import { drag } from 'd3-drag';
import { last, mapKeys, mapValues, sortBy } from 'lodash-es';

import { getters } from '/app/store';

let ICONS = mapValues(
	mapKeys(
		{
			...import.meta.glob('/assets/svgs/regular/*.svg', { eager: true }),
			...import.meta.glob('/assets/svgs/solid/*.svg', { eager: true }),
		},
		(_, k) => {
			return last(k.split('/')).split('.')[0];
		}
	),
	(v) => v.default
);

const MARGIN = 35;
const NODE_RADIUS = 15;
const NODE_ICON_FACTOR = 0.5;

// Colors

const COLOR_LINE_NORMAL = 'rgb(190, 190, 190)';
const COLOR_LINE_NORMAL_BG = 'rgb(90, 90, 90)';
const COLOR_LINE_COLLISION = 'red';
const COLOR_LINE_SELECTED = 'cornflowerblue';
const COLOR_LINE_SENDING = 'lightgreen';
const COLOR_LINE_RECEIVED = 'yellow';

// State

let state = {
	svg: null,
	container: null,
	onZoom: null,
	lastZoomTransform: null,
	lines: null,
	nodes: null,
	nodeLabels: null,
	nodeOverlay: null,
	xAxisGrid: null,
	yAxisGrid: null,
	xAxis: null,
	yAxis: null,
	zoom: null,
	handler: {},
	selected: null,
	map: {
		element: null,
		leaflet: null,
	},
};

// Helper

function getCoordsFromEvent(e, width, height) {
	let rect = e.target.getBoundingClientRect();

	let px = e.pageX - rect.left - MARGIN;
	let py = e.pageY - rect.top - MARGIN;

	if (px < 0 || py < 0 || px >= width || py >= height) return null;

	return {
		x: parseFloat(((px / width) * getters.config().kmRange - getters.config().origin.x).toFixed(3)),
		y: parseFloat(((py / height) * getters.config().kmRange - getters.config().origin.y).toFixed(3)),
	};
}

function getCoords(x, y, width, height) {
	if (x < 0 || y < 0 || x >= width || y >= height) return null;

	return {
		x: parseFloat(((x / width) * getters.config().kmRange - getters.config().origin.x).toFixed(3)),
		y: parseFloat(((y / height) * getters.config().kmRange - getters.config().origin.y).toFixed(3)),
	};
}

export function setupGrid(element, handler) {
	// Clear

	d3.select(element).selectAll('svg').remove();

	// Sizing

	let width = element.offsetWidth - MARGIN * 2;
	let height = element.offsetHeight - MARGIN * 2;

	// Map

	/*state.map.element = d3.select(element).append('div').attr('class', 'map absolute left-0 top-0 w-100 h-100 o-50').style('z-index', '-1');

	state.map.leaflet = L.map(state.map.element.node(), {
		zoomControl: false,
		zoomSnap: 0,
		zoom: {
			animate: false,
		},
		pan: {
			animate: false,
		},
	}).setView([49.8677109, 8.6521049], 14);

	// Disable all map interactions
	state.map.leaflet.dragging.disable();
	state.map.leaflet.touchZoom.disable();
	state.map.leaflet.doubleClickZoom.disable();
	state.map.leaflet.scrollWheelZoom.disable();
	state.map.leaflet.boxZoom.disable();
	state.map.leaflet.keyboard.disable();
	if (state.map.leaflet.tap) state.map.leaflet.tap.disable();

	L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
		attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
	}).addTo(state.map.leaflet);*/

	//
	// Base Element Setup
	//

	state.zoom = d3
		.zoom()
		.scaleExtent([1, 40])
		.extent([
			[MARGIN, MARGIN],
			[width + MARGIN, height + MARGIN],
		])
		.filter((e) => {
			e.preventDefault();
			return (!e.ctrlKey || e.type === 'wheel') && !e.button;
		})
		.on('zoom', (e) => {
			/*state.map.leaflet
				.setZoom(14 + (e.transform.k - 1))
				.setView([49.8677109, 8.6521049])
				.panBy([-(e.transform.x / 3), -(e.transform.y / 3)]);*/

			state.container.attr('transform', e.transform);
			if (state.onZoom) {
				state.onZoom(e);
			}
		});

	state.handler = handler ?? {};
	state.svg = d3
		.select(element)
		.append('svg')
		.style('width', `100%`)
		.style('height', `100%`)
		.on('click', (e) => {
			state.selected = null;
			if (state.handler.nodeClick) state.handler.nodeClick(e, null);
			update(element);
		})
		.on('contextmenu', (e) => {
			if (state.handler.context) state.handler.context(e, getCoordsFromEvent(e, width, height));
			e.preventDefault();
		})
		.on('mousemove', (e) => {
			if (state.handler.mouseMove) {
				state.handler.mouseMove(getCoordsFromEvent(e, width, height));
			}
		})
		.call(state.zoom);

	//
	// Create Definitions
	//

	let defs = state.svg.append('defs');

	let buildEndArrowMarker = (id, color) => {
		defs.append('svg:marker')
			.attr('id', id)
			.attr('markerWidth', 3)
			.attr('markerHeight', 3)
			.attr('refX', NODE_RADIUS * 0.45)
			.attr('refY', 1.5)
			.attr('orient', 'auto')
			.append('svg:polygon')
			.attr('points', '0 0, 3 1.5, 0 3')
			.attr('fill', color);
	};
	buildEndArrowMarker('arrow-end-collision', COLOR_LINE_COLLISION);
	buildEndArrowMarker('arrow-end-sending', COLOR_LINE_SENDING);
	buildEndArrowMarker('arrow-end-selected', COLOR_LINE_SELECTED);
	buildEndArrowMarker('arrow-end-normal', COLOR_LINE_NORMAL);
	buildEndArrowMarker('arrow-end-normal-bg', COLOR_LINE_NORMAL_BG);

	let buildStartArrowMarker = (id, color) => {
		defs.append('svg:marker')
			.attr('id', id)
			.attr('markerWidth', 3)
			.attr('markerHeight', 3)
			.attr('refX', -(NODE_RADIUS / 4))
			.attr('refY', 1.5)
			.attr('orient', 'auto')
			.append('svg:polygon')
			.attr('points', '3 0, 0 1.5, 3 3')
			.attr('fill', color);
	};
	buildStartArrowMarker('arrow-start-collision', COLOR_LINE_COLLISION);
	buildStartArrowMarker('arrow-start-sending', COLOR_LINE_SENDING);
	buildStartArrowMarker('arrow-start-selected', COLOR_LINE_SELECTED);
	buildStartArrowMarker('arrow-start-normal', COLOR_LINE_NORMAL);
	buildStartArrowMarker('arrow-start-normal-bg', COLOR_LINE_NORMAL_BG);

	defs.append('svg:clipPath').attr('id', 'clip').append('svg:rect').attr('width', width).attr('height', height).attr('x', 0).attr('y', 0);

	//
	// Create Axis
	//

	state.xAxis = state.svg.append('g').attr('transform', `translate(${MARGIN}, ${MARGIN})`);
	state.yAxis = state.svg.append('g').attr('transform', `translate(${MARGIN}, ${MARGIN})`);

	state.xAxisGrid = state.svg
		.append('g')
		.attr('transform', `translate(${MARGIN}, ${MARGIN})`)
		.attr('class', 'axis-grid')
		.classed('background', getters.backgroundPresent());
	state.yAxisGrid = state.svg
		.append('g')
		.attr('transform', `translate(${MARGIN}, ${MARGIN})`)
		.attr('class', 'axis-grid')
		.classed('background', getters.backgroundPresent());

	//
	// Create Container for node, labels and lines
	//

	state.container = state.svg.append('g').attr('transform', `translate(${MARGIN}, ${MARGIN})`).attr('clip-path', 'url(#clip)').append('g');

	state.lines = state.container.append('g');
	state.nodes = state.container.append('g');
	state.nodeLabels = state.container.append('g');
	state.nodeOverlay = state.container.append('g');

	//
	// Background
	//

	d3.select('#sim-bg').style('background-size', '100% 100%');

	update(element);
}

export function update(element) {
	let width = element.offsetWidth - MARGIN * 2;
	let height = element.offsetHeight - MARGIN * 2;

	// Set class if background is present
	state.xAxisGrid.classed('background', getters.backgroundPresent());
	state.yAxisGrid.classed('background', getters.backgroundPresent());

	// Create linear scales
	let x = d3.scaleLinear().domain([0, getters.config().kmRange]).range([0, width]);
	let y = d3.scaleLinear().domain([0, getters.config().kmRange]).range([0, height]);

	//
	// Axis Setup
	//

	let offsetX = getters.config().origin.x;
	let offsetY = getters.config().origin.y;

	let axisBottom = d3.axisBottom(x);
	let axisLeft = d3.axisLeft(y);
	let axisGridBottom = d3
		.axisBottom(x)
		.tickSize(width)
		.tickFormat(() => '')
		.ticks(10);

	let axisGridRight = d3
		.axisRight(y)
		.tickSize(width)
		.tickFormat(() => '')
		.ticks(10);

	//
	// Zoom
	//

	let scaleByTransform = (val, transform) => {
		if (!transform && state.lastZoomTransform) {
			transform = state.lastZoomTransform.transform;
		}
		if (!transform) {
			return val;
		}
		return val / transform.k;
	};

	let updateSimBgZoom = ({ transform }) => {
		d3.select('#sim-bg')
			.style('background-position', `left ${transform.x}px top ${transform.y}px`)
			.style('background-size', `${transform.k * 100}% ${transform.k * 100}%`);
	};

	let updateAxisZoom = ({ transform }) => {
		state.xAxis.call(axisBottom.scale(transform.rescaleX(x)));
		state.yAxis.call(axisLeft.scale(transform.rescaleY(y)));
		state.xAxisGrid.call(axisGridBottom.scale(transform.rescaleX(x)));
		state.yAxisGrid.call(axisGridRight.scale(transform.rescaleY(y)));
	};

	let updateNodesZoom = ({ transform }) => {
		state.nodes
			.selectAll('circle')
			.attr('r', scaleByTransform(NODE_RADIUS, transform))
			.attr('stroke-width', (n) => {
				return state.selected === n.id ? scaleByTransform(2) : 0;
			});
		state.nodes
			.selectAll('image')
			.attr('x', (n) => x(n.x + offsetX) - scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR))
			.attr('y', (n) => y(n.y + offsetY) - scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR))
			.attr('width', scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR * 2))
			.attr('height', scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR * 2));
		state.nodeLabels
			.selectAll('text')
			.attr('font-size', scaleByTransform(13, transform) + 'px')
			.attr('x', (n) => x(n.x + offsetX) + scaleByTransform(NODE_RADIUS + 4))
			.attr('y', (n) => y(n.y + offsetY) - scaleByTransform(5));
	};

	let updateLinesZoom = ({ transform }) => {
		state.lines.selectAll('line').attr('stroke-width', scaleByTransform(3, transform));
	};

	state.onZoom = (e) => {
		state.lastZoomTransform = e;

		updateAxisZoom(e);
		updateNodesZoom(e);
		updateLinesZoom(e);
		updateSimBgZoom(e);
	};

	state.xAxis.attr('transform', `translate(${MARGIN}, ${MARGIN + height})`).call(axisBottom);
	state.yAxis.call(axisLeft);
	state.xAxisGrid.call(axisGridBottom);
	state.yAxisGrid.call(axisGridRight);

	if (state.lastZoomTransform) {
		updateAxisZoom(state.lastZoomTransform);
		updateNodesZoom(state.lastZoomTransform);
		updateLinesZoom(state.lastZoomTransform);
		updateSimBgZoom(state.lastZoomTransform);
	}

	//
	// Nodes
	//

	let setNode = (node) => {
		let nodeDragged = null;

		node.attr('cx', (n) => x(n.x + offsetX))
			.attr('cy', (n) => y(n.y + offsetY))
			.attr('r', scaleByTransform(NODE_RADIUS))
			.attr('stroke-width', (n) => {
				return state.selected === n.id ? scaleByTransform(2) : 0;
			})
			.attr('stroke', (n) => {
				return state.selected === n.id ? COLOR_LINE_SELECTED : 'none';
			})
			.style('fill', (n) => {
				if (getters.nodeStateById(n.id, 'collision') > 0) return COLOR_LINE_COLLISION;
				if (getters.nodeStateById(n.id, 'sending') > 0) return COLOR_LINE_SENDING;
				if (getters.nodeStateById(n.id, 'received') > 0) return COLOR_LINE_RECEIVED;
				return 'black';
			})
			.style('cursor', 'pointer')
			.on('contextmenu', (e, d) => {
				if (state.handler.nodeContext) state.handler.nodeContext(e, d.id);
				e.preventDefault();
				e.stopPropagation();
			})
			.on('click', (e, d) => {
				state.selected = d.id;
				update(element);
				if (state.handler.nodeClick) state.handler.nodeClick(e, d.id);
				e.stopPropagation();
			})
			.call(
				drag()
					.filter((e) => {
						return !e.ctrlKey && !e.button && e.shiftKey;
					})
					.clickDistance(0)
					.on('start', (e) => {
						nodeDragged = state.nodeOverlay
							.append('circle')
							.attr('cx', x(e.subject.x + offsetX) + e.dx)
							.attr('cy', y(e.subject.y + offsetY) + e.dy)
							.attr('r', scaleByTransform(NODE_RADIUS))
							.style('fill', 'yellow');
					})
					.on('drag', (e) => {
						nodeDragged = nodeDragged
							.attr('cx', parseFloat(nodeDragged.attr('cx')) + e.dx)
							.attr('cy', parseFloat(nodeDragged.attr('cy')) + e.dy);
					})
					.on('end', (e) => {
						if (state.handler.nodeDragged)
							state.handler.nodeDragged(
								e.subject,
								getCoords(parseFloat(nodeDragged.attr('cx')) + e.dx, parseFloat(nodeDragged.attr('cy')) + e.dy, width, height)
							);
						nodeDragged.remove();
					})
			);
	};

	let setNodeLabel = (node) => {
		node.attr('x', (n) => x(n.x + offsetX) + scaleByTransform(NODE_RADIUS + 4))
			.attr('y', (n) => y(n.y + offsetY) - scaleByTransform(5))
			.attr('font-size', scaleByTransform(13) + 'px')
			.style('pointer-events', 'none')
			.style('fill', (n) => {
				if (n.id === state.selected) return 'black';

				return 'rgba(0, 0, 0, 0.5)';
			})
			.text((n) => n.id);
	};

	let setNodeIcon = (node) => {
		node.attr('x', (n) => x(n.x + offsetX) - scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR))
			.attr('y', (n) => y(n.y + offsetY) - scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR))
			.attr('width', scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR * 2))
			.attr('height', scaleByTransform(NODE_RADIUS * NODE_ICON_FACTOR * 2))
			.attr('href', (n) => ICONS[n.icon])
			.style('pointer-events', 'none')
			.style('filter', 'invert(100%) sepia(0%) saturate(0%) hue-rotate(265deg) brightness(105%) contrast(102%)')
			.text((n) => n.id);
	};

	state.nodes
		.selectAll('circle')
		.data(getters.nodes(), (n) => n.id)
		.join(
			(enter) => enter.append('circle').call(setNode),
			(update) => setNode(update),
			(exit) => exit.remove()
		);

	state.nodes
		.selectAll('image')
		.data(
			getters.nodes().filter((n) => n.icon.length > 0),
			(n) => n.id
		)
		.join(
			(enter) => enter.append('image').call(setNodeIcon),
			(update) => setNodeIcon(update),
			(exit) => exit.remove()
		);

	state.nodeLabels
		.selectAll('text')
		.data(getters.nodes(), (n) => n.id)
		.join(
			(enter) => enter.append('text').call(setNodeLabel),
			(update) => setNodeLabel(update),
			(exit) => exit.remove()
		);

	//
	// Lines
	//

	let setLine = (line) => {
		line.attr('x1', (l) => x(l.a.x + offsetX))
			.attr('x2', (l) => x(l.b.x + offsetX))
			.attr('y1', (l) => y(l.a.y + offsetY))
			.attr('y2', (l) => y(l.b.y + offsetY))
			.attr('stroke-width', scaleByTransform(3))
			.attr('marker-start', (l) => {
				if (!l.aToB) return '';

				if (state.selected && (l.a.id === state.selected || l.b.id === state.selected)) return 'url(#arrow-start-selected)';

				if (getters.nodeStateById(l.b.id, 'collision') > 0) return 'url(#arrow-start-collision)';
				if (getters.nodeStateById(l.b.id, 'sending') > 0) return 'url(#arrow-start-sending)';

				if (getters.backgroundPresent()) return 'url(#arrow-start-normal-bg)';

				return 'url(#arrow-start-normal)';
			})
			.attr('marker-end', (l) => {
				if (!l.bToA) return '';

				if (state.selected && (l.a.id === state.selected || l.b.id === state.selected)) return 'url(#arrow-end-selected)';

				if (getters.nodeStateById(l.a.id, 'collision')) return 'url(#arrow-end-sending)';
				if (getters.nodeStateById(l.a.id, 'sending')) return 'url(#arrow-end-collision)';

				if (getters.backgroundPresent()) return 'url(#arrow-end-normal-bg)';

				return 'url(#arrow-end-normal)';
			})
			.attr('stroke', (l) => {
				if (state.selected && (l.a.id === state.selected || l.b.id === state.selected)) return COLOR_LINE_SELECTED;

				if (getters.nodeStateById(l.a.id, 'collision') > 0 || getters.nodeStateById(l.b.id, 'collision') > 0) return COLOR_LINE_COLLISION;
				if (getters.nodeStateById(l.a.id, 'sending') > 0 || getters.nodeStateById(l.b.id, 'sending') > 0) return COLOR_LINE_SENDING;

				if (getters.backgroundPresent()) return COLOR_LINE_NORMAL_BG;

				return COLOR_LINE_NORMAL;
			})
			.attr('fill', 'red');
	};

	// Sort lines to put selected on top
	let lines = getters.reachLines();
	let sortedLines = sortBy(lines, (val) => {
		if (state.selected !== null && (val.a.id === state.selected || val.b.id === state.selected)) {
			return 1;
		}
		return 0;
	});

	state.lines
		.selectAll('line')
		.data(sortedLines /*, (l) => l.key*/)
		.join(
			(enter) => enter.append('line').call(setLine),
			(update) => setLine(update),
			(exit) => exit.remove()
		);
}
