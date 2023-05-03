import { throttle } from 'lodash-es';
import naive from 'naive-ui';
import PubSub from 'pubsub-js';
import JsonViewer from 'vue-json-viewer';
import VueSimpleContextMenu from 'vue-simple-context-menu';
import { VueWinBox } from 'vue-winbox';
import { createApp, watch } from 'vue/dist/vue.esm-bundler';

import 'leaflet/dist/leaflet.css';
import 'tachyons/css/tachyons.css';
import 'vue-simple-context-menu/dist/vue-simple-context-menu.css';
import 'winbox/dist/css/themes/white.min.css';

import '/assets/css/index.css';

import * as API from '/app/api';
import * as Components from '/app/components';
import { setupGrid, update } from '/app/grid';
import { getters, store } from '/app/store';
import * as Windows from '/app/windows';
import '/app/ws';

let emptyNode = (base) => {
	return {
		...{
			id: 'Node ' + Math.ceil(Math.random() * 100),
			online: true,
			x: 0,
			y: 0,
			z: 1,
			txGain: 0,
			rxSens: 0,
		},
		...base,
	};
};

setInterval(() => {
	document.getElementById('running-for').innerText = Math.floor((Date.now() - getters.config().startTime) / 1000).toString();
}, 1000);

document.onkeypress = (e) => {
	PubSub.publish('KeyPress.' + e.code, e);
};

let app = createApp({
	data() {
		return {
			selectedId: null,

			// TX/RX pre-sets
			txRxPresets: [
				{ label: 'TTGO T-Beam V1.1 SX1276', value: '0', settings: [14, -148] },
				{ label: 'TTGO LORA32 V 2.0 @29mA', value: '1', settings: [13, -118] },
				{ label: 'TTGO LORA32 V 2.0 @90mA', value: '2', settings: [17, -118] },
				{ label: 'TTGO LORA32 V 2.0 @120mA', value: '3', settings: [20, -118] },
			],

			// node edit & creation
			editNode: emptyNode(),
			editNodeRules: {
				id: {
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid Node ID',
				},
				x: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid X value',
				},
				y: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid Y value',
				},
				z: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid Z value',
				},
				rxGain: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid RX Gain value',
				},
				txSens: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid TX Sensitivity value',
				},
			},
			showEdit: false,
			showCreate: false,

			// node multiple creation
			createMultiple: {
				idPrefix: 'Node',
				amount: 10,
				rxGain: 0,
				txSens: 0,
			},
			createMultipleRules: {
				idPrefix: {
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid Node ID prefix',
				},
				amount: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid amount value',
				},
				rxGain: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid RX Gain value',
				},
				txSens: {
					type: 'number',
					required: true,
					trigger: ['blur', 'input'],
					message: 'Please input a valid TX Sensitivity value',
				},
			},
			showCreateMultiple: false,
		};
	},
	mounted() {
		console.log('[Vue] Mounted');

		setupGrid(this.$refs.sim, {
			nodeClick: (_, id) => {
				// Note: If we directly call selectNode we will invoke a vue update in the updating of the grid.
				// This will lead to a bad delay, so we update the selection on the next tick.
				setTimeout(() => this.selectNode(id), 0);
			},
			nodeContext: (e, n) => {
				this.nodeContextOpen(e, getters.nodeById(n));
			},
			nodeDragged: (n, coords) => {
				if (!coords) return;

				API.updateNode({ ...n, ...coords })
					.then(() => console.log('[API] Update successful'))
					.catch(console.log);
			},
			context: (e, point) => {
				this.simContextOpen(e, point);
			},
		});

		// Throttle the grid re-renders to max every 33 ms, which is around 30fps
		let updateGrid = throttle(
			() => {
				// Don't update if not active
				if (document.hidden) return;

				update(this.$refs.sim);
			},
			33,
			{ leading: true, trailing: true }
		);

		document.addEventListener('visibilitychange', updateGrid);
		window.addEventListener('resize', updateGrid);
		watch(store, updateGrid);
	},
	updated() {
		console.log('[Vue] Updated');
	},
	methods: {
		selectNode(id) {
			this.selectedId = id;
		},
		nodeContextOpen(event, n) {
			this.nodeIsDraggingStart = false;
			this.$refs.nodeContext.showMenu(event, n);
		},
		nodeOptionClicked({ item, option }) {
			switch (option.name) {
				case 'Edit':
					this.showEdit = true;
					this.editNode = { ...item };

					this.$refs.editNodeWindow.winbox.setTitle('Update Node: ' + item.id);
					this.$refs.editNodeWindow.winbox.show();
					break;
				case 'Delete':
					API.deleteNode(item.id)
						.then(() => {
							if (this.selectedId === item.id) {
								this.selectedId = null;
							}
							console.log('[API] Delete successful');
						})
						.catch(console.log);
					break;
			}
		},
		simContextOpen(event, item) {
			this.$refs.simContext.showMenu(event, item);
		},
		simOptionClicked({ item, option }) {
			switch (option.name) {
				case 'Create Node':
					this.showCreate = true;
					this.editNode = emptyNode({ x: item.x, y: item.y });

					this.$refs.editNodeWindow.winbox.setTitle('Create Node');
					this.$refs.editNodeWindow.winbox.show();
					break;
				case 'Create Multiple Nodes':
					this.showCreateMultiple = true;
					this.createMultiple = {
						idPrefix: 'Node',
						amount: 10,
						rxGain: 0,
						txSens: 0,
					};

					this.$refs.createMultipleWindow.winbox.setTitle('Create Multiple Nodes');
					this.$refs.createMultipleWindow.winbox.show();
					break;
			}
		},
		createMultipleNodes() {
			this.$refs.createMultipleNodeForm.validate((errors) => {
				if (!errors) {
					Promise.all(
						new Array(this.createMultiple.amount).fill(null).map((_, i) => {
							let num = i + this.nodes.length;

							return API.createNode(
								emptyNode({
									id: this.createMultiple.idPrefix + num,
									txGain: this.createMultiple.txGain,
									rxSens: this.createMultiple.rxSens,
									x: Math.random() * (this.config.kmRange * 0.9) - this.config.origin.x,
									y: Math.random() * (this.config.kmRange * 0.9) - this.config.origin.y,
								})
							);
						})
					)
						.then(() => {
							this.showCreateMultiple = false;

							this.$refs.createMultipleWindow.winbox.hide();
						})
						.catch(console.log);
				} else {
				}
			});
		},
		createNode() {
			this.$refs.createNodeForm.validate((errors) => {
				if (!errors) {
					API.createNode(this.editNode)
						.then(() => {
							this.showCreate = false;
							this.showEdit = false;

							this.$refs.editNodeWindow.winbox.hide();
						})
						.catch(console.log);
				} else {
				}
			});
		},
		updateNode() {
			this.$refs.createNodeForm.validate((errors) => {
				if (!errors) {
					API.updateNode(this.editNode)
						.then(() => {
							this.showCreate = false;
							this.showEdit = false;

							this.$refs.editNodeWindow.winbox.hide();
						})
						.catch(console.log);
				} else {
				}
			});
		},
		closeCreateNode() {
			this.showCreate = false;
			this.showEdit = false;

			this.$refs.editNodeWindow.winbox.hide();

			return true;
		},
		closeMultipleNode() {
			this.showCreateMultiple = false;

			this.$refs.createMultipleWindow.winbox.hide();

			return true;
		},
	},
	computed: {
		selected() {
			if (this.selectedId === null) return null;
			return getters.nodeById(this.selectedId);
		},
		...getters,
	},
})
	.use(naive)
	.component('JsonViewer', JsonViewer)
	.component('VueWinBox', VueWinBox)
	.component('VueSimpleContextMenu', VueSimpleContextMenu);

// Register components and windows
Object.keys(Components).forEach((k) => app.component(k, Components[k]));
Object.keys(Windows).forEach((k) => app.component(k, Windows[k]));

app.mount('#app');
