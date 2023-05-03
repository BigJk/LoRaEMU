<template>
	<vue-win-box ref="window" :options="options">
		<div class="overflow-auto pa3 flex-grow-1 black-90" ref="log" style="font-family: monospace">
			<div v-for="e in events" class="mb1">
				<div class="mb1">
					<b>[{{ e.time }}]</b> {{ e.type }}
				</div>
				<json-viewer :value="e.data" :expand-depth="0"></json-viewer>
			</div>
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
				title: 'Events',
				class: ['white', 'no-close', 'no-max', 'no-full'],
				x: '50px',
				y: '90px',
				width: '450px',
				min: true,
				top: '51px',
			},
		};
	},
	updated() {
		this.$refs.log.scrollTop = this.$refs.log.scrollHeight;
	},
	computed: {
		...getters,
	},
};
</script>

<style scoped></style>
