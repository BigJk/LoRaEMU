package emu

import (
	"errors"
	"github.com/BigJk/loraemu/mobility"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

// Mobility represents a mobility manager that executes a set of mobility commands from
// the ns-2 format on a LoRaEMU instance.
type Mobility struct {
	emu         *Emulator
	tickrate    float64
	timeScaling int
	loop        bool
	commands    []mobility.Command
	wg          sync.WaitGroup
	done        chan bool
	pause       atomic.Value
}

// NewMobility creates a new mobility manager linked to an emu with the given commands.
func NewMobility(emu *Emulator, commands []mobility.Command) *Mobility {
	pause := atomic.Value{}
	pause.Store(false)

	return &Mobility{
		emu:         emu,
		commands:    commands,
		tickrate:    10,
		done:        make(chan bool, 1),
		pause:       pause,
		timeScaling: 1,
	}
}

// SetTickrate sets the ticks per seconds rate at which the sub-steps of movement
// will be calculated. 10 means 10 sub-steps per second, 30 means 30 sub-steps per second
// and so on.
func (m *Mobility) SetTickrate(tickrate float64) *Mobility {
	m.tickrate = tickrate
	return m
}

// SetTimeScaling ets the mobility run with a time speedup. A value of 10 would mean that 1 second only takes 100ms.
func (m *Mobility) SetTimeScaling(value int) error {
	if value <= 0 {
		return errors.New("scaling must be positive")
	}

	if value > 1000 {
		return errors.New("scaling can't be over 1000")
	}

	m.timeScaling = value

	return nil
}

// SetLoop changes if the mobility simulation should restart after finishing. Default is false.
func (m *Mobility) SetLoop(val bool) *Mobility {
	m.loop = val
	return m
}

func (m *Mobility) setInitialPositions() []*mobility.SetDestCommand {
	var dests []*mobility.SetDestCommand

	// set initial positions and convert from m to km
	for i := range m.commands {
		if m.commands[i].Set != nil {
			_ = m.emu.UpdateNode(m.commands[i].Set.Node, func(node *Node) error {
				switch m.commands[i].Set.Axis {
				case mobility.XAxis:
					node.X = m.commands[i].Set.Val / 1000.0
				case mobility.YAxis:
					node.Y = m.commands[i].Set.Val / 1000.0
				case mobility.ZAxis:
					node.Z = m.commands[i].Set.Val / 1000.0
				}

				return nil
			})
		} else if m.commands[i].SetDest != nil {
			setDest := *m.commands[i].SetDest
			setDest.X /= 1000.0
			setDest.Y /= 1000.0
			setDest.Speed /= 1000.0

			dests = append(dests, &setDest)
		}
	}

	// sort dests so that it is in order of time
	sort.SliceStable(dests, func(i, j int) bool {
		return dests[i].Time < dests[j].Time
	})

	return dests
}

func (m *Mobility) SetPause(val bool) {
	m.pause.Store(val)
}

func (m *Mobility) GetPause() bool {
	return m.pause.Load().(bool)
}

// Start starts the simulation.
func (m *Mobility) Start() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		for i := 0; i < 1 || m.loop; i++ {
			// set initial positions and get set destination commands
			dests := m.setInitialPositions()

			// abort if no dests exist
			if len(dests) == 0 {
				break
			}

			active := map[string]*mobility.SetDestCommand{}
			ticker := time.NewTicker(time.Millisecond * time.Duration(1000.0/float64(m.timeScaling)/m.tickrate))
			elapsed := 0.0

		commandExec:
			for {
				select {
				case <-ticker.C:
					{
						if m.pause.Load().(bool) {
							continue
						}

						elapsed += 1000.0 / 1000.0 / m.tickrate

						if len(dests) > 0 {
							for i := 0; i < len(dests); i++ {
								if elapsed >= dests[i].Time {
									active[dests[i].Node] = dests[i]
									dests = append(dests[:i], dests[i+1:]...)
									i--
								}
							}
						}

						for nodeId, command := range active {
							if active != nil {
								// don't act on speed values that can't reach the destination
								if command.Speed <= 0 {
									delete(active, nodeId)
									continue
								}

								err := m.emu.UpdateNode(nodeId, func(node *Node) error {
									stepSize := command.Speed / m.tickrate

									diff := mgl64.Vec2([2]float64{command.X - node.X, command.Y - node.Y})
									if diff.Len() < stepSize {
										node.X = command.X
										node.Y = command.Y

										delete(active, nodeId)
									} else {
										step := diff.Normalize().Mul(stepSize)
										node.X += step.X()
										node.Y += step.Y()
									}

									return nil
								})

								if err != nil {
									delete(active, nodeId)
								}
							}
						}

						if len(active) == 0 && len(dests) == 0 {
							break commandExec
						}
					}
				case <-m.done:
					return
				}
			}
		}
	}()
}

// Stop requests the stop of the simulation. You need to .Wait() after this to ensure graceful shutdown.
func (m *Mobility) Stop() {
	m.done <- true
}

// Done waits for the simulation to finish.
func (m *Mobility) Done() {
	m.wg.Wait()
}
