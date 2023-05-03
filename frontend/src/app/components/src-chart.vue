<template>
	<div ref="chart"></div>
</template>

<script>
import * as d3 from 'd3';

import { getters } from '/app/store';

export default {
	name: 'src-chart',
	props: ['nodes'],
	data: () => {
		return {
			svg: null,
			container: null,
			xAxis: null,
			yAxis: null,
			legend: null,
			updater: null,
		};
	},
	mounted() {
		this.svg = d3.select(this.$refs.chart).append('svg');

		this.legend = this.svg.append('g');
		this.legend.append('circle').attr('cx', 0).attr('cy', 0).attr('r', '5').attr('fill', 'lightgreen');
		this.legend.append('text').attr('font-size', 12).attr('x', 10).attr('y', 4).text('Sent');

		this.legend.append('circle').attr('cx', 60).attr('cy', 0).attr('r', '5').attr('fill', 'gold');
		this.legend.append('text').attr('font-size', 12).attr('x', 70).attr('y', 4).text('Received');

		this.legend.append('circle').attr('cx', 145).attr('cy', 0).attr('r', '5').attr('fill', 'red');
		this.legend.append('text').attr('font-size', 12).attr('x', 155).attr('y', 4).text('Collision');

		this.container = this.svg.append('g').attr('transform', 'translate(90, 0)');
		this.xAxis = this.container.append('g');
		this.yAxis = this.container.append('g');

		this.updateChart();
		this.updater = setInterval(this.updateChart, 1000);
	},
	beforeUnmount() {
		clearInterval(this.updater);
	},
	methods: {
		updateChart() {
			console.log('SRC Chart :: Update!');

			let width = this.$refs.chart.clientWidth;
			let height = this.$props.nodes.length * 20 + 40;

			this.svg.style('width', '100%').attr('height', height);

			this.legend.attr('transform', 'translate(50,' + (height - 15) + ')');

			height -= 50;

			let x = d3
				.scaleLinear()
				.domain([0, 100])
				.range([0, width - 100]);

			this.xAxis
				.attr('transform', 'translate(0,' + height + ')')
				.call(d3.axisBottom(x))
				.selectAll('text')
				.attr('transform', 'translate(-10,0) rotate(-45)')
				.style('text-anchor', 'end');

			let y = d3
				.scaleBand()
				.range([0, height])
				.domain(
					this.$props.nodes.map(function (d) {
						return d.id;
					})
				)
				.padding(0.1);

			this.yAxis.call(d3.axisLeft(y)).attr('transform', 'translate(-1,0)');

			let setSendingBar = (bar) => {
				bar.attr('class', 'bar-sending')
					.attr('x', x(0))
					.attr('y', function (d) {
						return y(d.id);
					})
					.attr('width', function (node) {
						let stats = getters.nodeStats()[node.id];
						if (!stats) {
							return x(0);
						}

						let sum = stats.sending + stats.collision + stats.received;
						if (sum === 0) return 0;

						return Math.max(x((stats.sending / sum) * 100), 0);
					})
					.attr('height', y.bandwidth())
					.attr('fill', 'lightgreen');
			};

			let setReceivedBar = (bar) => {
				bar.attr('class', 'bar-received')
					.attr('x', (node) => {
						let stats = getters.nodeStats()[node.id];
						if (!stats) {
							return x(0);
						}

						let sum = stats.sending + stats.collision + stats.received;
						if (sum === 0) return 0;

						return x((stats.sending / sum) * 100);
					})
					.attr('y', function (d) {
						return y(d.id);
					})
					.attr('width', function (node) {
						let stats = getters.nodeStats()[node.id];
						if (!stats) {
							return x(0);
						}

						let sum = stats.sending + stats.collision + stats.received;
						if (sum === 0) return 0;

						return Math.max(x((stats.received / sum) * 100), 0);
					})
					.attr('height', y.bandwidth())
					.attr('fill', 'gold');
			};

			let setCollisionBar = (bar) => {
				bar.attr('class', 'bar-collision')
					.attr('x', (node) => {
						let stats = getters.nodeStats()[node.id];
						if (!stats) {
							return x(0);
						}

						let sum = stats.sending + stats.collision + stats.received;
						if (sum === 0) return 0;

						return x((stats.sending / sum) * 100 + (stats.received / sum) * 100);
					})
					.attr('y', function (d) {
						return y(d.id);
					})
					.attr('width', function (node) {
						let stats = getters.nodeStats()[node.id];
						if (!stats) {
							return x(0);
						}

						let sum = stats.sending + stats.collision + stats.received;
						if (sum === 0) return 0;

						return Math.max(x((stats.collision / sum) * 100), 0);
					})
					.attr('height', y.bandwidth())
					.attr('fill', 'red');
			};

			this.container
				.selectAll('.bar-sending')
				.data(this.$props.nodes)
				.join(
					(enter) => {
						enter.append('rect').call(setSendingBar);
					},
					(update) => setSendingBar(update),
					(exit) => exit.remove()
				);

			this.container
				.selectAll('.bar-received')
				.data(this.$props.nodes)
				.join(
					(enter) => {
						enter.append('rect').call(setReceivedBar);
					},
					(update) => setReceivedBar(update),
					(exit) => exit.remove()
				);

			this.container
				.selectAll('.bar-collision')
				.data(this.$props.nodes)
				.join(
					(enter) => {
						enter.append('rect').call(setCollisionBar);
					},
					(update) => setCollisionBar(update),
					(exit) => exit.remove()
				);
		},
	},
};
</script>

<style scoped></style>
