<template>
	<vue-win-box ref="window" :options="options">
		<div class="overflow-auto pa3 flex-grow-1 black-90">
			<n-tabs justify-content="space-evenly" type="line" size="small">
				<n-tab-pane name="SRC" tab="SRC">
					<n-alert class="mb3" title="Info" type="info">Represents the send / receive / collision ratio per node.</n-alert>
					<src-chart :nodes="nodes"></src-chart>
				</n-tab-pane>
				<n-tab-pane name="SRC Live Timeline" tab="SRC Live Timeline">
					<n-alert class="mb3" title="Info" type="info">Represents the send / receive / collision ratio of all nodes live.</n-alert>
					<live-src-chart></live-src-chart>
				</n-tab-pane>
			</n-tabs>
		</div>
	</vue-win-box>
</template>

<script>
import { getters } from '/app/store';

export default {
	name: 'window-events',
	data: () => {
		return {
			options: {
				title: 'Statistics',
				class: ['white', 'no-close', 'no-max', 'no-full'],
				x: '50px',
				y: '90px',
				width: '650px',
				min: true,
				top: '51px',
			},
			updater: null,
			currentSRCData: null,
		};
	},
	computed: {
		nodes: getters.nodes,
	},
};
</script>

<style scoped></style>
