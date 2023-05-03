#!/bin/bash

rm -r ./release/
mkdir ./release/
mkdir ./release/frontend

ECHO Building LoRaEMU...
(cd ./cmd/emu && go build) || { echo 'Failed!' ; exit 1; }
ECHO Done!

ECHO Building LoRaEMU Inspector...
(cd ./cmd/log-inspect && go build) || { echo 'Failed!' ; exit 1; }
ECHO Done!

ECHO Building LoRaEMU Frontend...
(cd ./frontend && npm i && npm run build) || { echo 'Failed!' ; exit 1; }
ECHO Done!

mv ./cmd/emu/emu ./release/emu
mv ./cmd/log-inspect/log-inspect ./release/log-inspect
mv ./frontend/dist ./release/frontend/