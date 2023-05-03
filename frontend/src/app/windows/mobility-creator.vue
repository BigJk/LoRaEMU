<template>
	<vue-win-box ref="window" :options="options">
		<div class="overflow-auto pa3 flex-grow-1 black-90">
			<div class="mb2 flex justify-between">
				<div><b>Node:</b> {{ selectedId ?? 'none selected' }}</div>
				<div>
					<div class="mb2"><b>Start:</b> x={{ start[0] }} / y={{ start[1] }}</div>
				</div>
			</div>
			<n-input-number v-model:value="speed" class="mb2" clearable>
				<template #suffix> m/s </template>
			</n-input-number>
			<n-checkbox v-model:checked="alternate" class="mb2"> Alternate </n-checkbox>
			<n-input v-model:value="nsWaypoints" type="textarea" placeholder="ns2 file..." class="mb2" :rows="17" />
			<div class="flex justify-between">
				<n-button :on-click="this.removeLast">Remove Last</n-button>
				<n-button :on-click="this.reset">Reset</n-button>
			</div>
		</div>
	</vue-win-box>
</template>

<script>
import PubSub from 'pubsub-js';

import { getters } from '/app/store';

export default {
	name: 'mobility-creator',
	props: ['selectedId'],
	data() {
		return {
			options: {
				title: 'Mobility Creator',
				class: ['white', 'no-close', 'no-max', 'no-full'],
				x: '50px',
				y: '90px',
				width: '500px',
				height: '600px',
				min: true,
				top: '51px',
				hidden: false,
			},
			subs: [],
			speed: 20,
			startOffset: 0,
			alternate: false,
			start: [0, 0],
			waypoints: [],
		};
	},
	mounted() {
		this.subs.push(
			PubSub.subscribe('KeyPress.KeyM', () => {
				if (this.$refs.window.hidden || this.selected === null) return;

				this.waypoints.push([this.selected.x * 1000, this.selected.y * 1000]);
			})
		);

		this.subs.push(
			PubSub.subscribe('KeyPress.KeyS', () => {
				if (this.$refs.window.hidden || this.selected === null) return;

				this.start = [this.selected.x * 1000, this.selected.y * 1000];
			})
		);
	},
	beforeUnmount() {
		this.subs.forEach(PubSub.unsubscribe);
	},
	methods: {
		removeLast() {
			this.waypoints.splice(this.waypoints.length - 1, 1);
		},
		reset() {
			this.start = [0, 0];
			this.waypoints = [];
		},
	},
	computed: {
		selected() {
			if (this.selectedId === null) return null;
			return getters.nodeById(this.$props.selectedId);
		},
		nsWaypoints() {
			if (this.selected === null) return '';

			let curTime = this.startOffset;
			let curPos = [this.start[0], this.start[0]];
			let nsPoints = [];

			this.waypoints.forEach((w, i) => {
				let dist = Math.sqrt(Math.pow(curPos[0] - w[0], 2) + Math.pow(curPos[1] - w[1], 2));
				let time = dist / this.speed;

				nsPoints.push(`$ns_ at ${curTime.toFixed(4)} ${'$' + this.selectedId} setdest ${w[0]} ${w[1]} ${this.speed}`);
				curTime += time;
				curPos = w;
			});

			if (this.alternate) {
				[...this.waypoints.slice().reverse().slice(1), this.start].forEach((w, i) => {
					let dist = Math.sqrt(Math.pow(curPos[0] - w[0], 2) + Math.pow(curPos[1] - w[1], 2));
					let time = dist / this.speed;

					nsPoints.push(`$ns_ at ${curTime.toFixed(4)} ${'$' + this.selectedId} setdest ${w[0]} ${w[1]} ${this.speed}`);
					curTime += time;
				});
			}

			return (
				`${'$' + this.selectedId} set X_ ${this.start[0]}
${'$' + this.selectedId} set Y_ ${this.start[1]}
` + nsPoints.join('\n')
			);
		},
	},
};
</script>

<style scoped></style>
