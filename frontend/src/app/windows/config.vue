<template>
	<vue-win-box ref="window" :options="options">
		<div class="pa3 black-90">
			<div class="b f5 mb3">Path Loss</div>

			<div>Gamma</div>
			<n-slider class="mv2" v-model:value="config.gamma" :onUpdateValue="updateReachLines" :step="0.1"></n-slider>
			<n-input-number
				v-model:value="config.gamma"
				@focus="updateReachLines"
				@blur="updateReachLines"
				:onUpdateValue="updateReachLines"
				size="small"
			></n-input-number>

			<div class="mt3">Ref. Distance</div>
			<n-slider class="mv2" v-model:value="config.refDist" :onUpdateValue="updateReachLines" :step="0.1"></n-slider>
			<n-input-number
				v-model:value="config.refDist"
				@focus="updateReachLines"
				@blur="updateReachLines"
				:onUpdateValue="updateReachLines"
				size="small"
			></n-input-number>

			<div class="b f5 mv3">Nodes</div>

			<n-input :value="JSON.stringify(this.sortedNodes, null, 2)" type="textarea" rows="15"></n-input>
		</div>
	</vue-win-box>
</template>

<script>
import { sortBy } from 'lodash-es';

import { getters, mutations } from '/app/store';

export default {
	name: 'config',
	data: () => {
		return {
			options: {
				title: 'Configs',
				class: ['white', 'no-close', 'no-max', 'no-full'],
				x: 'center',
				y: 'center',
				width: '400px',
				min: true,
				top: '51px',
			},
		};
	},
	methods: {
		updateReachLines() {
			mutations.updateReachLines();
		},
	},
	computed: {
		sortedNodes() {
			return sortBy(this.nodes, ['id']);
		},
		...getters,
	},
};
</script>

<style scoped></style>
