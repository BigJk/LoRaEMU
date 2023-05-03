package emu

import (
	"encoding/json"
	"errors"
	"github.com/BigJk/loraemu/lora"
	"io"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

type Event string

const (
	EventNodeAdded           = Event("NodeAdded")
	EventNodeRemoved         = Event("NodeRemoved")
	EventNodeUpdated         = Event("NodeUpdated")
	EventCollision           = Event("NodeCollision")
	EventSending             = Event("NodeSending")
	EventReceived            = Event("NodeReceived")
	EventPayloadSizeExceeded = Event("NodePayloadSizeExceeded")
)

const (
	// CollisionDecodeableLevel describes the dB strength a received packets needs to have compared
	// to other packets it's collides with. If this is the case the packet can be decoded
	// even while collision.
	CollisionDecodeableLevel = 6

	// MaxPacketLen specifies the maximum length a single LoRa packet can be.
	MaxPacketLen = 255
)

// LogEntry represents an entry in the trace log of the emulator.
type LogEntry struct {
	Time   time.Time `json:"time"`
	Event  Event     `json:"event"`
	NodeID string    `json:"nodeId"`
	Data   any       `json:"data"`
}

// RxPacket represents a received packet with its corresponding signal information.
type RxPacket struct {
	RSSI     int     `json:"rssi"`
	SNR      int     `json:"snr"`
	Data     []byte  `json:"data"`
	RecvTime int64   `json:"recvTime"`
	Airtime  float64 `json:"airtime"`
}

type OnReceivedFn func(node Node, packet RxPacket)
type OnEventFn func(event Event, node Node, data any)

// Emulator represents a LoRa emulator.
type Emulator struct {
	sync.RWMutex
	sync.WaitGroup

	freq             float64
	gamma            float64
	refDist          float64
	kmRange          float64
	ignoreCollisions bool
	timeScaling      int
	packetConfig     lora.PacketConfig
	nodes            map[string]Node
	onReceived       OnReceivedFn
	onEvent          OnEventFn
	snrOffset        int

	startTime int64

	traceMutex sync.Mutex
	trace      io.Writer
	logger     logr.Logger
}

// New creates a new emulator with the given frequency, gamma (which is the Log-Distance Path Loss exponent)
// and LoRa packet config.
func New(freq float64, gamma float64, refDist float64, kmRange float64, config lora.PacketConfig) *Emulator {
	return &Emulator{
		freq:         freq,
		gamma:        gamma,
		refDist:      refDist,
		kmRange:      kmRange,
		timeScaling:  1,
		packetConfig: config,
		nodes:        map[string]Node{},
		onReceived:   func(node Node, packet RxPacket) {},
		onEvent:      func(event Event, node Node, data any) {},
		startTime:    time.Now().UnixMilli(),
		logger:       logr.Discard(),
	}
}

func (emu *Emulator) GetFreq() float64 {
	return emu.freq
}

func (emu *Emulator) GetRefDist() float64 {
	return emu.refDist
}

func (emu *Emulator) GetGamma() float64 {
	return emu.gamma
}

func (emu *Emulator) GetKMRange() float64 {
	return emu.kmRange
}

func (emu *Emulator) GetStartTime() int64 {
	return emu.startTime
}

// SetTraceWriter sets the writer for the trace logs. If no writer was set no trace logs will be emitted.
func (emu *Emulator) SetTraceWriter(writer io.Writer) {
	emu.Lock()
	defer emu.Unlock()

	emu.trace = writer
}

// SetLogger sets the logger. This will log additional information that are not relevant for the trace.
func (emu *Emulator) SetLogger(logger logr.Logger) {
	emu.Lock()
	defer emu.Unlock()

	emu.logger = logger
}

// SetOnReceived sets the callback that should be called if a simulated node receives a message.
func (emu *Emulator) SetOnReceived(onReceived OnReceivedFn) {
	emu.Lock()
	defer emu.Unlock()

	emu.onReceived = onReceived
}

// SetOnEvent sets the callback that should be called if a event happens in the simulator.
func (emu *Emulator) SetOnEvent(onEvent OnEventFn) {
	emu.Lock()
	defer emu.Unlock()

	emu.onEvent = onEvent
}

// SetIgnoreCollision enables or disables the collision detection.
func (emu *Emulator) SetIgnoreCollision(state bool) {
	emu.Lock()
	defer emu.Unlock()

	emu.ignoreCollisions = state
}

// SetTimeScaling (warning: experimental!) lets the simulator run with a time speedup. A value of 10 would mean that 1 second only takes 100ms.
func (emu *Emulator) SetTimeScaling(value int) error {
	if value <= 0 {
		return errors.New("scaling must be positive")
	}

	if value > 1000 {
		return errors.New("scaling can't be over 1000")
	}

	emu.Lock()
	defer emu.Unlock()

	emu.timeScaling = value

	return nil
}

// SetSNROffset sets a static offset that will be added to the RSSI and the node SNR (SNR = RSSI + Node.SNR + SNROffset).
func (emu *Emulator) SetSNROffset(value int) {
	emu.Lock()
	defer emu.Unlock()

	emu.snrOffset = value
}

// NodeIDs returns all the node ids as strings.
func (emu *Emulator) NodeIDs() []string {
	emu.RLock()
	defer emu.RUnlock()

	var ids []string
	for k := range emu.nodes {
		ids = append(ids, k)
	}

	return ids
}

// Nodes returns all the Nodes as a copy.
func (emu *Emulator) Nodes() []Node {
	emu.RLock()
	defer emu.RUnlock()

	var nodes []Node

	for _, v := range emu.nodes {
		nodes = append(nodes, v)
	}

	return nodes
}

// HasNode checks if a node with the given id exists.
func (emu *Emulator) HasNode(id string) bool {
	emu.RLock()
	defer emu.RUnlock()

	_, ok := emu.nodes[id]
	return ok
}

// GetNode gets a node by id.
func (emu *Emulator) GetNode(id string) Node {
	emu.RLock()
	defer emu.RUnlock()

	return emu.nodes[id]
}

// AddNode adds a node to the simulation.
func (emu *Emulator) AddNode(node Node) error {
	if err := node.Valid(); err != nil {
		return err
	}

	emu.Lock()
	defer emu.Unlock()

	if _, ok := emu.nodes[node.ID]; ok {
		return errors.New("already exists")
	}

	emu.nodes[node.ID] = node
	emu.emitEvent(EventNodeAdded, node, nil)

	return nil
}

// UpdateNode updates a node by a given id. The updater function will be called with the node
// and any changes to the node in that function will be set to the emulator.
//
// Don't call any other emulator functions in the updater to avoid deadlocks!
func (emu *Emulator) UpdateNode(id string, updater func(node *Node) error) error {
	emu.Lock()
	defer emu.Unlock()

	if _, ok := emu.nodes[id]; !ok {
		return errors.New("not found")
	}

	selectedNode := emu.nodes[id]
	if err := updater(&selectedNode); err != nil {
		return err
	}

	selectedNode.ID = id // prevent update to id

	if err := selectedNode.Valid(); err != nil {
		return err
	}

	emu.nodes[id] = selectedNode

	emu.emitEvent(EventNodeUpdated, selectedNode, nil)

	return nil
}

// RemoveNode removes a node by id.
func (emu *Emulator) RemoveNode(id string) error {
	emu.Lock()
	defer emu.Unlock()

	node, ok := emu.nodes[id]
	if !ok {
		return errors.New("not found")
	}

	delete(emu.nodes, id)
	emu.emitEvent(EventNodeRemoved, node, nil)

	return nil
}

// Clear removes all nodes.
func (emu *Emulator) Clear() {
	emu.Lock()
	defer emu.Unlock()

	emu.nodes = map[string]Node{}
}

func (emu *Emulator) getTime() time.Time {
	elapsed := time.Now().UnixMilli() - emu.startTime
	elapsed *= int64(emu.timeScaling)
	return time.UnixMilli(emu.startTime + elapsed)
}

func (emu *Emulator) emitEvent(event Event, node Node, data any) {
	emu.onEvent(event, node, data)

	// TODO: error handling
	if emu.trace != nil {
		bytes, _ := json.Marshal(LogEntry{
			Time:   emu.getTime(),
			Event:  event,
			NodeID: node.ID,
			Data:   data,
		})

		emu.traceMutex.Lock()
		defer emu.traceMutex.Unlock()

		_, _ = emu.trace.Write(bytes)
		_, _ = emu.trace.Write([]byte{'\n'})
	}
}

// SendMessage starts the data sending for a given node by id.
func (emu *Emulator) SendMessage(id string, msg []byte) error {
	emu.Lock()
	defer emu.Unlock()

	if _, ok := emu.nodes[id]; !ok {
		return errors.New("not found")
	}

	sender := emu.nodes[id]
	if !sender.Online {
		return errors.New("sender not online")
	}

	packet := emu.packetConfig
	packet.PayloadLen = float64(len(msg))

	// Deny packets that are too long
	if len(msg)+int(emu.packetConfig.PreambleLen) >= MaxPacketLen {
		emu.emitEvent(EventPayloadSizeExceeded, sender, map[string]interface{}{
			"size":                len(msg) + int(emu.packetConfig.PreambleLen),
			"theoretical_airtime": packet.TimeTotal(),
		})

		return nil
	}

	start := emu.getTime().UnixMilli()
	stop := start + int64(packet.TimeTotal())

	// We are already sending and need to wait for the sent to finish.
	if start <= sender.sendingUntil {
		emu.Add(1)
		go func() {
			wait := (1000 * float64(sender.sendingUntil-start+1)) / float64(emu.timeScaling)
			time.Sleep(time.Microsecond * time.Duration(wait))
			_ = emu.SendMessage(id, msg)
			emu.Done()
		}()

		return nil
	}

	// Set the sending until
	sender.sendingUntil = stop
	emu.nodes[id] = sender

	emu.emitEvent(EventSending, sender, map[string]interface{}{
		"start":   start,
		"stop":    stop,
		"airtime": packet.TimeTotal(),
		"x":       sender.X,
		"y":       sender.Y,
		"z":       sender.Z,
	})

	for k, receiver := range emu.nodes {
		if k == id || !receiver.Online {
			continue
		}

		reachedGain := sender.TXGain - sender.PathLoss(receiver, emu.refDist, emu.gamma, emu.freq)
		if reachedGain > receiver.RXSens {
			emu.logger.Info("sending", "from", id, "to", k, "gain", reachedGain, "margin", reachedGain-receiver.RXSens, "dist", sender.DistanceTo(receiver))

			r := received{
				Start: start,
				Stop:  stop,
				Gain:  reachedGain,
			}

			receiver.receiving = append(receiver.receiving, r)
			emu.nodes[k] = receiver

			emu.Add(1)
			go func(id string, sleep float64, gain float64, timeFrame received, msg []byte) {
				defer emu.Done()

				time.Sleep(time.Microsecond * time.Duration(1000*sleep))

				emu.RLock()

				node, ok := emu.nodes[id]
				if !ok {
					return
				}

				emu.RUnlock()

				collisions := 0

				// node is sending itself and can't receive at the same time
				if timeFrame.Start <= node.sendingUntil {
					collisions++
				}

				// check if multiple packets arrive
				for i := range node.receiving {
					if timeFrame.Start >= node.receiving[i].Start && timeFrame.Start <= node.receiving[i].Stop && gain-node.receiving[i].Gain < CollisionDecodeableLevel {
						collisions += 1
					}
				}

				if emu.ignoreCollisions || collisions <= 1 {
					packet := RxPacket{
						RSSI:     int(gain),
						SNR:      int(gain) + node.SNR + emu.snrOffset,
						Data:     msg,
						RecvTime: emu.getTime().Unix(),
						Airtime:  sleep,
					}
					emu.onReceived(node, packet)
					emu.emitEvent(EventReceived, node, packet)
				} else {
					emu.emitEvent(EventCollision, node, packet)
				}
			}(k, packet.TimeTotal()/float64(emu.timeScaling), reachedGain, r, msg)
		}
	}

	return nil
}
