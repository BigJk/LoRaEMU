# LoRaEMU

```
  _        ___      ___ __  __ _   _ 
 | |   ___| _ \__ _| __|  \/  | | | |
 | |__/ _ \   / _` | _|| |\/| | |_| |
 |____\___/_|_\__,_|___|_|  |_|\___/
------------------------------------------
Daniel Schmidt <info@daniel-schmidt.me>

:: LoRaEMU is a simple LoRa simulator using Log-Distance Path Loss,
:: Collision detection, NS-2 Mobility File parsing and running.

USAGE:
  -config string
        specifies which file to load the config from. (default "./config.json")
  -debug
        sets debug mode.
  -log string
        specifies where to store the logs. the file will be overwritten! (default "./logs.txt")
  -timeout string
        specifies if the emulator should shut down after a certain amount of time (e.g. 1m, 1h20m, 50s, ...). If not specified run infinitely.
```

This is the main LoRaEMU executable and contains the emulator and webserver hosting the live view and a REST API to further control the simulation.

## Getting Started

To run the example configuration run:

```
emu --config=./example_config.json -log=./logs.txt
```

- Make sure that the ``frontend/dist`` exits with the bundled frontend. See the README.md in the Frontend folder for more information.

## Building

- You need to have [go](https://go.dev/) (at least ``1.18``) installed 
- Run ``go build``

## Config File

The LoRaEMU expects a json encoded config file with the following values:

```json5
{
  "freq": 868, // frequency LoRa operates on
  "gamma": 2.7, // the Log-Distance Path Loss exponent
  "refDistance": 0.1, // the Log-Distance reference distance in km
  
  "kmRange": 10, // the area in km that the live web-view will show
  
  // LoRa config to calculate airtime
  "packetConfig": {
    "preambleLen": 6,
    "spreadingFactor": 7,
    "bandWidth": 125,
    "codingRate": 8,
    "crc": true,
    "explicitHeader": false,
    "lowDataRateOptimization": false
  },
  
  // array of nodes that are initially placed in the simulation
  "nodes": [
    {
      "id": "Node1",
      "online": true,
      "x": 2,
      "y": 2,
      "z": 1,
      "txGain": 20,
      "rxSens": -139,
      "snr": 0 // constant snr value that will be returned for the node
    }
  ],
  
  // ns-2 mobility file that should be run on the nodes
  "mobility": {
    "file": "./mobility_example.ns2",
    "tickrate": 20, // specifies how many sub-steps per seconds are calculated
    "loop": true // if the movement should be restarted after finishing
  },
  
  // bind address of webserver
  "web": ":8291"
}
```

## Debug Mode

The debug mode is only needed when the frontend is run in development mode. If debug mode is enabled the webserver of the emulator will pass the appropriate requests to the frontend dev server.

See the README.md in the frontend folder for more information.

# Web

The webserver contains a frontend that shows a live view of the simulation and exposes a variety of API routes that can be used to fetch node content and dynamically change nodes.

## Live View

To access the frontend open ``http://127.0.0.1:[PORT]``, where the port is specified by the value in the ``web`` field of the config.

## API Routes

- All response bodies will be JSON encoded.
- All POST or PATCH request expect JSON encoded data with the correct ``application/json`` MIME type.

### Get Nodes: ``(GET) /api/nodes``

- Gets all nodes returned as array of node objects.

### Get Node IDs: ``(GET) /api/node_ids``

- Gets all node ids returned as an array of strings.

### Get Node: ``(GET) /api/node/:id``

- Gets a node by id.
- Returned as node object.

### Get Node Lat Lng: ``(GET) /api/node/:id/latlng``

- Gets a node lat and long values by id.
- Returned a array with 2 elements ``[lat, lng]``.

### Create Node: ``(POST) /api/node/create``

- Creates a node.
- Expects the request body to contain a node object.

### Update Node: ``(PUT) /api/node/update``

- Updates a node.
- Expects the request body to contain a node object.
- The id in the node object specifies which node to update.

### Delete Node: ``(DELETE) /api/node/:id``

- Deletes the node by id.

## Websocket

The most important part about LoRaEMU is the websocket communication. If we want to send and receive LoRa packets from a certain node we need to connect to that node over websocket.


### Node: ``(WS) /api/emu/:id``

- Websocket connection for a given node id
- If you want to send packets just send byte arrays
- Received packets will be JSON encoded RxPackets

**RxPacket**
```json5
{
  "rssi": -29, // Signal strength that the antenna received the packet with
  "snr": 0, // Signal-to-noise ratio from the node config
  "data": "dGVzdA==", // Bas64 encoded packet data
  "recvTime": 1670494949 // Unix timestamp of received time
}
```