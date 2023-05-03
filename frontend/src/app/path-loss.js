import { sortBy } from 'lodash-es';

export function canReach(config, a, b) {
	if (a.id === b.id) return false;

	let fspl = (distance) => 20 * Math.log10(distance) + 20 * Math.log10(config.freq) + 32.45;
	let logpl = (distance) => fspl(config.refDist) + 10 * config.gamma * Math.log10(distance / config.refDist);
	let distance = Math.sqrt(Math.pow(a.x - b.x, 2) + Math.pow(a.y - b.y, 2) + Math.pow(a.z - b.z, 2)) * 1000;

	return a.txGain - logpl(distance) > b.rxSens;
}

export function reachLines(config, nodes) {
	let res = {};

	for (let i = 0; i < nodes.length; i++) {
		for (let j = 0; j < nodes.length; j++) {
			let order = sortBy([nodes[i], nodes[j]], ['id']);
			let key = order[0].id + '-' + order[1].id;

			// we only want to visit each pair once
			if (res[key]) continue;

			// calculate the rotation of the line start and ends
			let diff = [order[1].x - order[0].x, order[1].y - order[0].y];
			let rad = -Math.atan2(diff[0], diff[1]);

			let conn = {
				key: key,
				a: order[0],
				b: order[1],
				offset: [Math.cos(rad), Math.sin(rad)],
				aToB: canReach(config, order[0], order[1]),
				bToA: canReach(config, order[1], order[0]),
			};

			// only store when at least one connection exist
			if (conn.aToB || conn.bToA) {
				res[key] = conn;
			}
		}
	}

	return Object.keys(res).map((k) => res[k]);
}
