import PubSub from 'pubsub-js';
import { reactive } from 'vue';

import * as API from '/app/api';
import { reachLines } from '/app/path-loss';
import Worker from '/app/store-worker?worker';

export let store = reactive({
	config: {
		freq: 868,
		gamma: 2.5,
		refDist: 0.1,
		kmRange: 10,
		startTime: Date.now(),
		origin: {
			x: 0,
			y: 0,
		},
	},
	backgroundPresent: false,
	nodes: [],
	nodeState: {},
	nodeStats: {},
	reachLines: [],
	events: [],
	mobility: {
		available: false,
		paused: false,
	},
});

// Helper Functions

function addNodeStates(n) {
	store.nodeState[n.id] = {
		sending: 0,
		collision: 0,
		received: 0,
	};
}

function addNodeStats(n) {
	let emptyStats = {
		sending: 0,
		collision: 0,
		received: 0,
		timeline: {
			sending: [],
			collision: [],
			received: [],
		},
	};

	if (store.nodeStats[n.id]) {
		store.nodeStats[n.id] = {
			...emptyStats,
			...store.nodeStats[n.id],
		};
	} else {
		store.nodeStats[n.id] = emptyStats;
	}
}

function checkMobility() {
	API.getEmuPause()
		.then((res) => {
			if (!res.ok) {
				throw res;
			}

			res.json().then((val) => {
				store.mobility.available = true;
				store.mobility.paused = val;
			});
		})
		.catch(() => {
			store.mobility.available = false;
			console.log('[Store] No mobility!');
		});
}

function checkBackground() {
	let img = new Image();
	img.onload = () => (store.backgroundPresent = true);
	img.onerror = () => (store.backgroundPresent = false);
	img.src = '/api/background';
}

// Store Getter

export const getters = {
	config: () => store.config,
	nodes: () => store.nodes,
	nodeById: (id) => store.nodes.find((n) => n.id === id),
	nodeIndex: (id) => store.nodes.findIndex((n) => n.id === id),
	nodeState: () => store.nodeState,
	nodeStats: () => store.nodeStats,
	nodeStateById: (id, type) => {
		if (store.nodeState[id] === undefined) return 0;
		return store.nodeState[id][type];
	},
	reachLines: () => store.reachLines,
	events: () => store.events,
	mobility: () => store.mobility,
	backgroundPresent: () => store.backgroundPresent,
};

// Event Worker
//
// Note: Modern browsers will throttle inactive tabs, which will cause setTimeout and setInterval to not
// get updated with correct timing. We can avoid that by moving our event loop to a WebWorker, that essentially
// replaces the behaviour of setTimeout.

let eventWorker = new Worker();
let events = {};

eventWorker.onmessage = (e) => {
	// We want to run all the functions that the WebWorker sent, as the timeouts have been reached.
	e.data.forEach((id) => {
		events[id]();
		delete events[id];
	});
};

function queueEvent(func, length) {
	// We give our event a unique id and pass it to the WebWorker with our wait time in ms.
	let id = Math.ceil(Math.random() * 1000000) + '-' + Math.ceil(Math.random() * 1000000);
	events[id] = func;
	eventWorker.postMessage({ id, length });
}

// Store Mutations

export const mutations = {
	setNodes: (nodes) => {
		store.nodes = nodes;
		store.nodes.forEach(addNodeStates);
		store.nodes.forEach(addNodeStats);
	},
	updateNode: (node) => {
		if (getters.nodeIndex(node.id) === -1) {
			store.nodes.push(node);
			addNodeStates(node);
			addNodeStats(node);
			return;
		}

		store.nodes[getters.nodeIndex(node.id)] = { ...getters.nodeById(node.id), ...node };
	},
	removeNode: (id) => {
		store.nodes = store.nodes.filter((n) => n.id !== id);
		delete store.nodeState[id];
		delete store.nodeStats[id];
	},
	setConfig: (config) => {
		store.config = config;

		// Reset node state and set node stats to collected stats from the server
		store.nodeState = {};
		store.nodeStats = store.config.curNodeStats;
		Object.keys(store.nodeStats).forEach((id) => addNodeStats({ id }));

		// Run checks if mobility and background are present
		checkMobility();
		checkBackground();
	},
	addLog: (data, type) => {
		// Disabled log tracking for now.
		return;

		store.events.push({
			time: new Date().toLocaleString(),
			type: type,
			data: data,
		});
	},
	triggerNodeState: (id, state, length, time) => {
		// State
		store.nodeState[id][state] += 1;

		queueEvent(() => {
			store.nodeState[id][state] -= 1;
		}, length);

		// Stats
		store.nodeStats[id][state] += 1;
		store.nodeStats[id].timeline[state].push(time);
	},
	setMobilityPaused: (state) => {
		store.mobility.paused = state;
	},
	updateReachLines: () => {
		store.reachLines = reachLines(store.config, store.nodes);
	},
};

//
//
//
PubSub.subscribe('WebSocket', (topic, packet) => {
	switch (packet.event) {
		case 'Config':
			{
				console.log('[WebSocket] Simulator Config received');
				mutations.setConfig(packet);
				mutations.updateReachLines();
			}
			break;
		case 'Nodes':
			{
				mutations.setNodes(packet.nodes || []);
				mutations.updateReachLines();
			}
			break;
		case 'NodeUpdated':
			{
				mutations.updateNode(packet.node);
				mutations.updateReachLines();
			}
			break;
		case 'NodeSending':
			{
				mutations.addLog(packet, packet.event);
				mutations.triggerNodeState(packet.node.id, 'sending', Math.ceil(packet.data.airtime), packet.time);
			}
			break;
		case 'NodeReceived':
			{
				mutations.addLog(packet, packet.event);
				mutations.triggerNodeState(packet.node.id, 'received', Math.ceil(/*packet.data.airtime*/ 500), packet.time);
			}
			break;
		case 'NodeCollision':
			{
				mutations.addLog(packet, packet.event);
				mutations.triggerNodeState(packet.node.id, 'collision', Math.ceil(/*packet.data.airtime*/ 500), packet.time);
			}
			break;
		case 'NodeRemoved':
			{
				mutations.addLog(packet, packet.event);
				mutations.removeNode(packet.node.id);
				mutations.updateReachLines();
			}
			break;
		case 'NodeAdded':
			{
				mutations.addLog(packet, packet.event);
				mutations.updateNode(packet.node);
				mutations.updateReachLines();
			}
			break;
	}
});
