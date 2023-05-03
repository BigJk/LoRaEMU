<template>
	<vue-win-box ref="window" :options="options">
		<div class="overflow-auto pa3 black-90">
			<n-button :type="mobility.paused ? 'success' : 'warning'" @click="togglePauseMobility">
				<template v-if="mobility.paused">Continue</template>
				<template v-else>Pause</template>
			</n-button>
		</div>
	</vue-win-box>
</template>

<script>
import * as API from '../api';
import { watch } from 'vue';

import { getters, mutations, store } from '/app/store';

export default {
	name: 'mobility',
	data: () => {
		return {
			options: {
				title: 'Mobility',
				class: ['white', 'no-close', 'no-max', 'no-full'],
				x: '50px',
				y: '90px',
				width: '300px',
				height: '100px',
				min: false,
				top: '51px',
				hidden: true,
			},
		};
	},
	mounted() {
		watch(store, () => {
			if (getters.mobility().available) {
				if (this.$refs.window.winbox.hidden) {
					this.$refs.window.winbox.show(true);
					this.$refs.window.winbox.minimize(true);
				}
			} else {
				this.$refs.window.winbox.hide(true);
				this.$refs.window.winbox.minimize(false);
			}
		});
	},
	methods: {
		togglePauseMobility() {
			API.setEmuPause(!this.mobility.paused)
				.then(() => {
					mutations.setMobilityPaused(!this.mobility.paused);

					this.$refs.window.winbox.setTitle('Mobility: ' + (this.mobility.paused ? 'Paused' : 'Running'));
				})
				.catch(console.log);
		},
	},
	computed: {
		...getters,
	},
};
</script>

<style scoped></style>
