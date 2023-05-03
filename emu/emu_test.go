package emu

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var timeScaling = []int{1, 10, 20, 50, 100}

// TestEmulator_Collision tests if in when two nodes send at the same time a collision is correctly detected.
// As two nodes send at the same time no node will receive a message because a node can't send and receive at the same time
// and the one that isn't sending will face a collision from the other too.
func TestEmulator_Collision(t *testing.T) {
	for _, scale := range timeScaling {
		t.Run(fmt.Sprintf("TimeScaling%d", scale), func(t *testing.T) {
			e := New(868, 10, 1, 10, lora.PacketConfigDefault)
			assert.NoError(t, e.SetTimeScaling(scale))

			assert.NoError(t, e.AddNode(Node{
				ID:     "1",
				Online: true,
				X:      1,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "2",
				Online: true,
				X:      1.2,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "3",
				Online: true,
				X:      1.4,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			gotCollision := 0

			e.SetOnEvent(func(event Event, node Node, data any) {
				if event == EventCollision {
					gotCollision++
				}

				if event == EventReceived {
					assert.Fail(t, "node received message that it shouldn't")
				}
			})

			assert.NoError(t, e.SendMessage("1", []byte(strings.Repeat("HELLO WORLD", 10))))
			assert.NoError(t, e.SendMessage("3", []byte(strings.Repeat("HELLO WORLD", 10))))

			e.Wait()

			assert.Equal(t, 4, gotCollision, "collisions not detected")
		})
	}
}

// TestEmulator_CollisionPowerLevel tests if in when two nodes send at the same time and one of the sends gains is high enough
// only one collision is emitted and the message with the higher gain is still detected.
func TestEmulator_CollisionPowerLevel(t *testing.T) {
	for _, scale := range timeScaling {
		t.Run(fmt.Sprintf("TimeScaling%d", scale), func(t *testing.T) {
			e := New(868, 10, 1, 10, lora.PacketConfigDefault)
			assert.NoError(t, e.SetTimeScaling(scale))

			assert.NoError(t, e.AddNode(Node{
				ID:     "1",
				Online: true,
				X:      1,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "2",
				Online: true,
				X:      1.2,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "3",
				Online: true,
				X:      1.4,
				Y:      1,
				Z:      0,
				TXGain: 80,
				RXSens: -200,
				SNR:    0,
			}))

			gotCollision := 0

			e.SetOnEvent(func(event Event, node Node, data any) {
				if event == EventCollision {
					gotCollision++
				}
			})

			assert.NoError(t, e.SendMessage("1", []byte(strings.Repeat("HELLO WORLD", 10))))
			assert.NoError(t, e.SendMessage("3", []byte(strings.Repeat("HELLO WORLD", 10))))

			e.Wait()

			assert.Equal(t, 3, gotCollision, "collisions not detected")
		})
	}
}

// TestEmulator_Collision tests if in when two nodes send at different times no collision is detected.
func TestEmulator_NoCollision(t *testing.T) {
	for _, scale := range timeScaling {
		t.Run(fmt.Sprintf("TimeScaling%d", scale), func(t *testing.T) {
			e := New(868, 10, 1, 10, lora.PacketConfigDefault)
			assert.NoError(t, e.SetTimeScaling(scale))

			assert.NoError(t, e.AddNode(Node{
				ID:     "1",
				Online: true,
				X:      1,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "2",
				Online: true,
				X:      1.2,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "3",
				Online: true,
				X:      1.4,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			gotCollision := false

			e.SetOnEvent(func(event Event, node Node, data any) {
				if node.ID == "2" && event == EventCollision {
					gotCollision = true
				}
			})

			assert.NoError(t, e.SendMessage("1", []byte(strings.Repeat("HELLO WORLD", 10))))

			time.Sleep(time.Millisecond * time.Duration(1000/scale))

			assert.NoError(t, e.SendMessage("3", []byte(strings.Repeat("HELLO WORLD", 10))))

			e.Wait()

			assert.False(t, gotCollision, "collision detected")
		})
	}
}

func TestEmulator_PayloadSizeExceeded(t *testing.T) {
	for _, scale := range timeScaling {
		t.Run(fmt.Sprintf("TimeScaling%d", scale), func(t *testing.T) {
			e := New(868, 10, 1, 10, lora.PacketConfigDefault)
			assert.NoError(t, e.SetTimeScaling(scale))

			assert.NoError(t, e.AddNode(Node{
				ID:     "1",
				Online: true,
				X:      1,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			gotExceeded := false

			e.SetOnEvent(func(event Event, node Node, data any) {
				if node.ID == "1" && event == EventPayloadSizeExceeded {
					gotExceeded = true
				}
			})

			assert.NoError(t, e.SendMessage("1", []byte(strings.Repeat("HELLO WORLD", 100))))

			e.Wait()

			assert.True(t, gotExceeded, "exceeding not detected")
		})
	}
}

func TestEmulator_MultipleSends(t *testing.T) {
	for _, scale := range timeScaling {
		t.Run(fmt.Sprintf("TimeScaling%d", scale), func(t *testing.T) {
			e := New(868, 10, 1, 10, lora.PacketConfigDefault)
			assert.NoError(t, e.SetTimeScaling(scale))

			assert.NoError(t, e.AddNode(Node{
				ID:     "1",
				Online: true,
				X:      1,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			assert.NoError(t, e.AddNode(Node{
				ID:     "2",
				Online: true,
				X:      1.3,
				Y:      1,
				Z:      0,
				TXGain: 40,
				RXSens: -200,
				SNR:    0,
			}))

			gotPacket := 0

			e.SetOnEvent(func(event Event, node Node, data any) {
				if node.ID == "2" && event == EventReceived {
					gotPacket++
				}
			})

			for i := 0; i < 20; i++ {
				assert.NoError(t, e.SendMessage("1", []byte(strings.Repeat("HELLO WORLD", 10))))
			}

			e.Wait()

			assert.Equal(t, 20, gotPacket, "didn't get all packages")
		})
	}
}

func BenchmarkEmulator_UpdateNode(b *testing.B) {
	e := New(868, 10, 1, 10, lora.PacketConfigDefault)
	assert.NoError(b, e.AddNode(Node{
		ID:     "1",
		Online: true,
		X:      1,
		Y:      1,
		Z:      0,
		TXGain: 40,
		RXSens: -200,
		SNR:    0,
	}))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = e.UpdateNode("1", func(node *Node) error {
			node.X = rand.Float64()
			node.Y = rand.Float64()

			return nil
		})
	}
}
