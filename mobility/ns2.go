package mobility

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

var setRegex = regexp.MustCompile(`\$(.+) set ([XYZ])_ ([\d\.]+)`)
var setDestRegex = regexp.MustCompile(`\$.+ at ([\d\.]+) "?\$(.+) setdest ([\d\.]+) ([\d\.]+) ([\d\.]+)"?`)

// Axis represents a target axis (X, Y, Z).
type Axis = byte

const (
	XAxis = Axis(0)
	YAxis = Axis(1)
	ZAxis = Axis(2)
)

// SetCommand represents the set command from ns-2, which sets an axis of a specified node to a value.
type SetCommand struct {
	Node string  `json:"node"`
	Axis Axis    `json:"axis"`
	Val  float64 `json:"val"`
}

// SetDestCommand represents the setdest movement command from ns-2, which starts a movement of a node
// to a specific destination.
type SetDestCommand struct {
	Node  string  `json:"node"`
	Time  float64 `json:"val"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Speed float64 `json:"speed"`
}

// Command represents an optional between SetCommand and SetDestCommand.
type Command struct {
	Set     *SetCommand     `json:"set"`
	SetDest *SetDestCommand `json:"setDest"`
}

// Parse a ns-2 mobility file from a reader.
func Parse(reader io.Reader) ([]Command, error) {
	scan := bufio.NewScanner(reader)
	lineCount := 1

	var commands []Command
	for scan.Scan() {
		line := scan.Text()

		match := setRegex.FindStringSubmatch(line)
		if len(match) > 0 {
			var set SetCommand

			set.Node = match[1]
			switch match[2] {
			case "X":
				set.Axis = XAxis
			case "Y":
				set.Axis = YAxis
			case "Z":
				set.Axis = ZAxis
			default:
				return nil, errors.New(fmt.Sprintf("unknown axis %s on line %d", match[2], lineCount))
			}

			val, err := strconv.ParseFloat(match[3], 64)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("can't parse float (%s)", match[3]))
			}

			set.Val = val

			commands = append(commands, Command{Set: &set})
		} else {
			match = setDestRegex.FindStringSubmatch(line)
			if len(match) > 0 {
				var setDest SetDestCommand

				time, err := strconv.ParseFloat(match[1], 64)
				if err != nil {
					return nil, errors.New(fmt.Sprintf("can't parse float (%s)", match[1]))
				}

				setDest.Time = time
				setDest.Node = match[2]

				x, err := strconv.ParseFloat(match[3], 64)
				if err != nil {
					return nil, errors.New(fmt.Sprintf("can't parse float (%s)", match[3]))
				}

				y, err := strconv.ParseFloat(match[4], 64)
				if err != nil {
					return nil, errors.New(fmt.Sprintf("can't parse float (%s)", match[4]))
				}

				speed, err := strconv.ParseFloat(match[5], 64)
				if err != nil {
					return nil, errors.New(fmt.Sprintf("can't parse float (%s)", match[5]))
				}

				setDest.X = x
				setDest.Y = y
				setDest.Speed = speed

				commands = append(commands, Command{SetDest: &setDest})
			}
		}

		lineCount++
	}

	return commands, nil
}
