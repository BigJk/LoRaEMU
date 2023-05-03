import { remove } from 'lodash-es';

let events = [];

setInterval(() => {
	if (events.length === 0) return;

	let now = performance.now();
	let toExecute = remove(events, (e) => now >= e.until);

	if (toExecute.length === 0) return;

	postMessage(toExecute.map((e) => e.id));
}, 10);

onmessage = (e) => {
	events.push({
		id: e.data.id,
		until: performance.now() + e.data.length,
	});
};
