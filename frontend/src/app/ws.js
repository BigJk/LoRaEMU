import PubSub from 'pubsub-js';
import ReconnectingWebSocket from 'reconnecting-websocket';

let conn = new ReconnectingWebSocket('ws://' + location.host + '/api/ws');

document.addEventListener('visibilitychange', () => {
	if (document.hidden) {
		console.log('[WebSocket] Closing WebSocket because of tab inactivity!');
		conn.close();
	} else {
		console.log('[Tab Active] Re-Open WebSocket because tab is active again!');
		conn.reconnect();
	}
});

conn.onopen = (e) => {
	console.log('[WebSocket] Open');
};

conn.onmessage = (e) => {
	let packet = JSON.parse(e.data);
	PubSub.publish('WebSocket.' + packet.event, packet);
};
