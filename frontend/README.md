# LoRaEMU Frontend

This is the Web-Frontend of LoRaEMU.

## Building

```
npm i
npm run build
```

This will create the ``dist`` folder which contains the bundled frontend. This folder needs to be present (e.g. copied over or symlinked) in the working directory of LoRaEMU for the emulator to be able to expose it.

## Development

```
npm i
npm run dev
```

This will start the vite dev-server and any changes to the frontend code will be hot-reloaded. If LoRaEMU is started with the debug flag ``--debug`` the emulator will use the dev-server as frontend and not the ``dist`` folder. This makes development really fast as you can see changes to the frontend live. 