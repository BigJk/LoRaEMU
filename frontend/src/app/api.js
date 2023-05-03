export function deleteNode(nodeId) {
	return fetch('/api/node/' + nodeId, {
		method: 'delete',
		headers: {
			'Content-Type': 'application/json',
		},
	});
}

export function getNode(nodeId) {
	return fetch('/api/node/' + nodeId, {
		method: 'get',
		headers: {
			'Content-Type': 'application/json',
		},
	});
}

export function createNode(node) {
	return fetch('/api/node/create', {
		method: 'post',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify(node),
	});
}

export function updateNode(node) {
	return fetch('/api/node/update', {
		method: 'put',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify(node),
	});
}

export function getEmuPause() {
	return fetch('/api/emu/pause', {
		method: 'get',
		headers: {
			'Content-Type': 'application/json',
		},
	});
}

export function setEmuPause(val) {
	return fetch('/api/emu/pause', {
		method: 'post',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({
			state: val,
		}),
	});
}
