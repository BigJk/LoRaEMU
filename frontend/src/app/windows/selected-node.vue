<template>
	<vue-win-box ref="window" :options="options">
		<div class="overflow-auto pa3 lh-copy black-90">
			<div v-if="selected === null">No Node Selected...</div>
			<div v-else>
				<div class="bb b--black-10 mb3 pb3">
					<div><b class="dib w4">ID:</b> {{ selected.id }}</div>
					<div>
						<b class="dib w4">Lat:</b>
						{{ ((Math.acos(Math.sqrt(offsetX * offsetX + offsetY * offsetY) / 6371.0) * 180.0) / Math.PI).toFixed(4) }}
					</div>
					<div>
						<b class="dib w4">Lng:</b>
						{{ ((Math.atan2(offsetY, offsetX) * 180.0) / Math.PI).toFixed(4) }}
					</div>
					<div>
						<b class="dib w4">X:</b>
						{{ selected.x.toFixed(4) }}
					</div>
					<div>
						<b class="dib w4">Y:</b>
						{{ selected.y.toFixed(4) }}
					</div>
				</div>
				<div class="mb3">
					<div><b class="dib w4">Sent</b> {{ nodeStats[selected.id].sending }}</div>
					<div><b class="dib w4">Received:</b> {{ nodeStats[selected.id].received }}</div>
					<div><b class="dib w4">Collisions:</b> {{ nodeStats[selected.id].collision }}</div>
				</div>
				<div v-if="selected.meta && Object.keys(selected.meta).length > 0" class="bt pt3 b--black-10">
					<div v-for="(v, k) in selected.meta" class="mb2">
						<b class="dib w5">{{ k }}</b> <a @click.prevent="openUrl(v)" v-if="v.indexOf('http') === 0" :href="v">{{ v }}</a>
						<span v-else>{{ v }}</span>
					</div>
				</div>
			</div>
		</div>
	</vue-win-box>
</template>

<script>
import { useWinBox } from 'vue-winbox';

import RefreshSVG from '/assets/svgs/solid/rotate.svg';

import { getters } from '/app/store';

const createWindow = useWinBox();

export default {
	name: 'selected-node',
	props: ['selectedId'],
	data: () => {
		return {
			options: {
				title: 'Selected Node',
				class: ['white', 'no-close', 'no-max', 'no-full'],
				x: 'center',
				y: 'center',
				width: '450px',
				min: true,
				top: '51px',
			},
		};
	},
	watch: {
		selectedId: function () {
			if (this.$props.selectedId) {
				this.$refs.window.winbox.setTitle('Selected Node: ' + this.$props.selectedId);
			} else {
				this.$refs.window.winbox.setTitle('Selected Node');
			}
		},
	},
	methods: {
		openUrl(url) {
			let window = createWindow({
				title: `${this.$props.selectedId} - ${url}`,
				class: ['white', 'no-max', 'no-full'],
				x: 'center',
				y: 'center',
				url: url,
			});

			window.addControl({
				index: 0,
				class: 'wb-refresh',
				image: RefreshSVG,
				click: function (event, winbox) {
					// Force reload even with cross-domain
					winbox.window.querySelector('iframe').src += '';
				},
			});
		},
	},
	computed: {
		selected() {
			if (!this.$props.selectedId) return null;
			return getters.nodeById(this.$props.selectedId);
		},
		offsetX() {
			return this.selected.x + 100;
		},
		offsetY() {
			return this.selected.y + 100;
		},
		...getters,
	},
};
</script>

<style scoped></style>
