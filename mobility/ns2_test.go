package mobility

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	test := bytes.NewBufferString(`$node_(0) set X_ 150.0
$node_(0) set Y_ 93.98597018956875
$ns_ at 0.0 $node_(0) setdest 150.0 110.0 50.40378694202284
$ns_ at 0.3177148143422528 $node_(1) setdest 159.68580405978113 110.0 50.40378694196106
$ns_ at 0.5098790275378633 $node_(1) setdest 164.68580405978113 110.0 50.129146111974336`)

	correct := []Command{
		{
			Set: &SetCommand{
				Node: "node_(0)",
				Axis: XAxis,
				Val:  150,
			},
			SetDest: nil,
		},
		{
			Set: &SetCommand{
				Node: "node_(0)",
				Axis: YAxis,
				Val:  93.98597018956875,
			},
			SetDest: nil,
		},
		{
			Set: nil,
			SetDest: &SetDestCommand{
				Node:  "node_(0)",
				Time:  0.0,
				X:     150.0,
				Y:     110.0,
				Speed: 50.40378694202284,
			},
		},
		{
			Set: nil,
			SetDest: &SetDestCommand{
				Node:  "node_(1)",
				Time:  0.3177148143422528,
				X:     159.68580405978113,
				Y:     110.0,
				Speed: 50.40378694196106,
			},
		},
		{
			Set: nil,
			SetDest: &SetDestCommand{
				Node:  "node_(1)",
				Time:  0.5098790275378633,
				X:     164.68580405978113,
				Y:     110.0,
				Speed: 50.129146111974336,
			},
		},
	}

	cmds, err := Parse(test)
	if !assert.NoError(t, err) {
		return
	}

	assert.EqualValues(t, correct, cmds)
}
