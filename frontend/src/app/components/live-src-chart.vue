<template>
	<div ref="chart"></div>
</template>

<script>
import * as d3 from 'd3';
import { pickBy } from 'lodash-es';
import PubSub from 'pubsub-js';

// Specifies the second size of the bucket. Example: If this value is 2 samples will be
// collected in 2-second intervals, so 1 bucket contains how many sends, receives and collisions
// happened in a 2-second window.
const BUCKET_HISTORY_SECONDS = 5;

// How much of the history should be shown. Example: If this is 10 and the history seconds are 2
// 10 * 2 = 20 seconds of history will be shown.
const BUCKET_HISTORY_SIZE = 25;

let buckets = {};

export default {
	name: 'live-src-chart',
	data: () => {
		return {
			svg: null,
			container: null,
			areaSending: null,
			areaReceived: null,
			areaCollision: null,
			xAxis: null,
			yAxis: null,
			legend: null,
			updater: null,
		};
	},
	mounted() {
		this.svg = d3.select(this.$refs.chart).append('svg');
		this.container = this.svg.append('g').attr('transform', 'translate(50, 0)');
		this.areaCollision = this.container.append('path');
		this.areaReceived = this.container.append('path');
		this.areaSending = this.container.append('path');
		this.xAxis = this.container.append('g');
		this.yAxis = this.container.append('g');

		this.legend = this.svg.append('g');
		this.legend.append('circle').attr('cx', 0).attr('cy', 0).attr('r', '5').attr('fill', 'lightgreen');
		this.legend.append('text').attr('font-size', 12).attr('x', 10).attr('y', 4).text('Sent');

		this.legend.append('circle').attr('cx', 60).attr('cy', 0).attr('r', '5').attr('fill', 'gold');
		this.legend.append('text').attr('font-size', 12).attr('x', 70).attr('y', 4).text('Received');

		this.legend.append('circle').attr('cx', 145).attr('cy', 0).attr('r', '5').attr('fill', 'red');
		this.legend.append('text').attr('font-size', 12).attr('x', 155).attr('y', 4).text('Collision');

		this.updateChart();

		this.updater = setInterval(this.updateChart, 1000);
	},
	beforeUnmount() {
		clearInterval(this.updater);
	},
	methods: {
		updateChart() {
			let now = Math.floor(performance.now() / 1000 / BUCKET_HISTORY_SECONDS);
			let data = new Array(BUCKET_HISTORY_SIZE).fill({
				NodeSending: 0,
				NodeReceived: 0,
				NodeCollision: 0,
			});

			data.forEach((_, i) => {
				if (buckets[now - BUCKET_HISTORY_SIZE + i]) {
					data[i] = buckets[now - BUCKET_HISTORY_SIZE + i];
				}
			});

			let width = this.$refs.chart.clientWidth;
			let height = 350;

			this.legend.attr('transform', 'translate(50,' + (height - 15) + ')');

			this.svg.style('width', '100%').attr('height', height);

			height -= 60;

			let x = d3
				.scaleLinear()
				.range([0, width - 55])
				.domain([BUCKET_HISTORY_SIZE - 1, 0]);

			this.xAxis
				.attr('transform', 'translate(0,' + height + ')')
				.call(
					d3
						.axisBottom(x)
						.ticks(BUCKET_HISTORY_SIZE)
						.tickFormat((_, i) => i * BUCKET_HISTORY_SECONDS + 's')
				)
				.selectAll('text')
				.attr('transform', 'translate(-10,0) rotate(-45)')
				.style('text-anchor', 'end');

			let y = d3
				.scaleLinear()
				.domain([
					0,
					Math.max(
						10,
						Math.ceil(
							d3.max(data, (d) => {
								return d.NodeSending + d.NodeReceived + d.NodeCollision;
							}) * 1.2
						)
					),
				])
				.range([height, 20]);

			this.yAxis.call(d3.axisLeft(y)).attr('transform', 'translate(-1,0)');

			this.areaSending
				.datum(data)
				.attr('class', 'area-sending')
				.attr('fill', '#cce5df')
				.attr('stroke', '#69b3a2')
				.attr('stroke-width', 1.5)
				.attr(
					'd',
					d3
						.area()
						.x(function (d, i) {
							return x(i);
						})
						.y0(y(0))
						.y1(function (d, i) {
							let res = buckets[now - BUCKET_HISTORY_SIZE + i];
							if (res) {
								return y(res.NodeSending);
							}
							return y(0);
						})
				);

			this.areaReceived
				.datum(data)
				.attr('class', 'area-received')
				.attr('fill', '#d7d791')
				.attr('stroke', 'yellow')
				.attr('stroke-width', 1.5)
				.attr(
					'd',
					d3
						.area()
						.x(function (d, i) {
							return x(i);
						})
						.y0(y(0))
						.y1(function (d, i) {
							let res = buckets[now - BUCKET_HISTORY_SIZE + i];
							if (res) {
								return y(res.NodeSending + res.NodeReceived);
							}
							return y(0);
						})
				);

			this.areaCollision
				.datum(data)
				.attr('class', 'area-collision')
				.attr('fill', '#d79191')
				.attr('stroke', 'red')
				.attr('stroke-width', 1.5)
				.attr(
					'd',
					d3
						.area()
						.x(function (d, i) {
							return x(i);
						})
						.y0(y(0))
						.y1(function (d, i) {
							let res = buckets[now - BUCKET_HISTORY_SIZE + i];
							if (res) {
								return y(res.NodeSending + res.NodeReceived + res.NodeCollision);
							}
							return y(0);
						})
				);
		},
	},
};

// Tracking of Events

function trigger(timestamp, type) {
	if (buckets[timestamp] === undefined) {
		buckets[timestamp] = {
			NodeSending: 0,
			NodeReceived: 0,
			NodeCollision: 0,
		};
	}

	buckets[timestamp][type] += 1;
}

function onMessage(_, packet) {
	trigger(Math.floor(performance.now() / 1000 / BUCKET_HISTORY_SECONDS), packet.event);
}

PubSub.subscribe('WebSocket.NodeSending', onMessage);
PubSub.subscribe('WebSocket.NodeReceived', onMessage);
PubSub.subscribe('WebSocket.NodeCollision', onMessage);

// Periodically drop old samples that are not shown anymore.
setInterval(() => {
	buckets = pickBy(buckets, (_, k) => {
		let diff = performance.now() / 1000 / BUCKET_HISTORY_SECONDS - parseInt(k);
		return diff < BUCKET_HISTORY_SIZE + 1;
	});
}, BUCKET_HISTORY_SIZE * BUCKET_HISTORY_SECONDS * 1000);
</script>

<style scoped></style>
