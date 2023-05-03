package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/BigJk/loraemu/emu"
	"github.com/BigJk/loraemu/lora"
	"github.com/BigJk/loraemu/mobility"
	"github.com/BigJk/loraemu/server"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type CommandConfig struct {
	DelayMs     int      `json:"delayMs"`
	BeforeStart int      `json:"beforeStart"`
	PerNode     []string `json:"perNode"`
	Start       []string `json:"start"`
	End         []string `json:"end"`
}

type Config struct {
	Freq        float64 `json:"freq"`
	Gamma       float64 `json:"gamma"`
	RefDistance float64 `json:"refDistance"`
	KMRange     float64 `json:"kmRange"`
	Origin      struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"origin"`
	PacketConfig     lora.PacketConfig `json:"packetConfig"`
	IgnoreCollisions bool              `json:"ignoreCollisions"`
	SNROffset        int               `json:"snrOffset"`
	TimeScaling      int               `json:"timeScaling"`
	Nodes            []emu.Node        `json:"nodes"`
	Commands         CommandConfig     `json:"commands"`
	Mobility         struct {
		File     string  `json:"file"`
		Tickrate float64 `json:"tickrate"`
		Loop     bool    `json:"loop"`
	} `json:"mobility"`
	BackgroundImage string `json:"backgroundImage"`
	Web             string `json:"web"`
}

type RunningCommand struct {
	cmd     *exec.Cmd
	logFile *os.File
}

var logger = logrusr.New(logrus.New())
var runningCommands []RunningCommand
var specialFilter = regexp.MustCompile(`[^a-zA-Z0-9_ ]+`)

func stopAndHelp() {
	fmt.Println()
	printHelp()
	os.Exit(-1)
}

func printHelp() {
	fmt.Println("  _        ___      ___ __  __ _   _ \n | |   ___| _ \\__ _| __|  \\/  | | | |\n | |__/ _ \\   / _` | _|| |\\/| | |_| |\n |____\\___/_|_\\__,_|___|_|  |_|\\___/")
	fmt.Println("------------------------------------------")
	fmt.Printf("Daniel Schmidt <info@daniel-schmidt.me>\n\n")
	fmt.Printf(":: LoRaEMU is a simple LoRa simulator using Log-Distance Path Loss,\n:: Collision detection, NS-2 Mobility File parsing and running.\n\nUSAGE:\n")
	flag.PrintDefaults()
}

func loadConfig(path string) Config {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Error(err, "can't read config file")
		stopAndHelp()
	}

	var conf Config
	if err := json.Unmarshal(configBytes, &conf); err != nil {
		logger.Error(err, "can't parse config file")
		stopAndHelp()
	}

	return conf
}

func loadTraceLogsFile(path string) io.WriteCloser {
	logs, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		logger.Error(err, "can't open log file")
		stopAndHelp()
	}
	return logs
}

func evalCommand(cmd string, params map[string]interface{}) (string, []string, error) {
	functions := map[string]govaluate.ExpressionFunction{
		"fmt": func(args ...interface{}) (interface{}, error) {
			format := args[0].(string)
			return fmt.Sprintf(format, args[1:]...), nil
		},
		"env": func(args ...interface{}) (interface{}, error) {
			env := os.Getenv(args[0].(string))

			// check if default value is given as second param
			if len(args) > 1 && len(env) == 0 {
				env = fmt.Sprint(args[1])
			}

			return env, nil
		},
		"rand": func(args ...interface{}) (interface{}, error) {
			return (float64)(rand.Intn(int(args[0].(float64)))), nil
		},
		"rand_str": func(args ...interface{}) (interface{}, error) {
			letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
			strLen := int(args[0].(float64))
			str := make([]byte, strLen)

			for i := 0; i < strLen; i++ {
				str[i] = letters[rand.Intn(len(letters))]
			}

			return string(str), nil
		},
		"safe": func(args ...interface{}) (interface{}, error) {
			return specialFilter.ReplaceAllString(args[0].(string), ""), nil
		},
	}

	runCmd := ""

	// check if a relative path is given, so we skip evaluation
	if strings.HasPrefix(cmd, "./") {
		runCmd = cmd
	} else {
		expression, err := govaluate.NewEvaluableExpressionWithFunctions(cmd, functions)
		if err != nil {
			return "", nil, err
		}

		result, err := expression.Evaluate(params)
		if err != nil {
			return "", nil, err
		}

		evalCmd, ok := result.(string)
		if !ok {
			return "", nil, errors.New("expression didn't return a string")
		}

		runCmd = strings.Replace(evalCmd, "#", "'", -1)
	}

	runCmdSplit := strings.Fields(runCmd)

	// detect bash mode
	if len(runCmdSplit) > 2 && runCmdSplit[0] == "bash" && runCmdSplit[1] == "-c" {
		return "bash", []string{"-c", strings.Join(runCmdSplit[2:], " ")}, nil
	}

	// try to combine hyphened strings back into a single string for correct argument passing
	var final []string
	inMulti := false
	for i := 1; i < len(runCmdSplit); i++ {
		count := strings.Count(runCmdSplit[i], "'")
		hasContain := strings.Contains(runCmdSplit[i], "'")
		hasStop := strings.HasSuffix(runCmdSplit[i], "'")

		if count == 2 && hasContain && hasStop {
			final = append(final, runCmdSplit[i])
			continue
		}

		if hasContain && !hasStop {
			inMulti = true
			final = append(final, runCmdSplit[i]+" ")
			continue
		}

		if hasStop {
			inMulti = false
			final[len(final)-1] += runCmdSplit[i]
			continue
		}

		if inMulti {
			final[len(final)-1] += runCmdSplit[i] + " "
		} else {
			final = append(final, runCmdSplit[i])
		}
	}

	return runCmdSplit[0], final, nil
}

func waitFor(t string, s *server.Server, maxNodes int) {
	if len(t) > 0 {
		switch strings.ToLower(t) {
		// wait for the first node to be connected to
		case "first":
			for s.ConnectedNodes() < 1 {
				time.Sleep(time.Millisecond * 100)
			}
		// wait for all nodes to be connected to
		case "all":
			for s.ConnectedNodes() < maxNodes {
				time.Sleep(time.Millisecond * 100)
			}
		default:
			logger.Error(nil, "unknown wait_for option", "wait_for", t)
			stopAndHelp()
		}
	}
}

func main() {
	configFile := flag.String("config", "./config.json", "specifies which file to load the config from.")
	logFile := flag.String("log", "./logs.txt", "specifies where to store the trace logs. the file will be overwritten!")
	debug := flag.Bool("debug", false, "sets debug mode.")
	timeout := flag.String("timeout", "", "specifies if the emulator should shut down after a certain amount of time (e.g. 1m, 1h20m, 50s, ...). If not specified run infinitely.")
	waitForStr := flag.String("wait_for", "", "specifies if the emulator should wait for nodes to connect before starting the timeout. Options: first, all")
	skipCommands := flag.Bool("skip_cmds", false, "skips all commands that would normaly executed in the scenario")
	ignoreCollisions := flag.Bool("ignore_collisions", false, "disables the collision detection of the emulator")
	noMobility := flag.Bool("no_mobility", false, "disables the mobility of the emulator")
	flag.Parse()

	// load config and open log file
	configFolder := filepath.Dir(*configFile)
	config := loadConfig(*configFile)
	trace := loadTraceLogsFile(*logFile)

	defer trace.Close()

	// create emulator, set configs and add nodes
	e := emu.New(config.Freq, config.Gamma, config.RefDistance, config.KMRange, config.PacketConfig)
	e.SetTraceWriter(trace)
	e.SetLogger(logger)
	e.SetIgnoreCollision(config.IgnoreCollisions || *ignoreCollisions)
	e.SetSNROffset(config.SNROffset)
	if config.TimeScaling > 0 {
		if err := e.SetTimeScaling(config.TimeScaling); err != nil {
			panic(err)
		}
	}

	for _, n := range config.Nodes {
		if err := e.AddNode(n); err != nil {
			panic(err)
		}
	}

	// create frontend server based on emulator
	s := server.New(e)

	if len(config.BackgroundImage) > 0 {
		imgFile, err := os.Open(filepath.Join(configFolder, config.BackgroundImage))
		if err != nil {
			logger.Error(err, "image not found")
			stopAndHelp()
		}

		img, _, err := image.Decode(imgFile)
		_ = imgFile.Close()

		if err != nil {
			logger.Error(err, "image not parsed")
			stopAndHelp()
		}

		s.SetBackgroundImage(img)
	}

	// check if timeout is specified and start goroutine
	if len(*timeout) > 0 {
		dur, err := time.ParseDuration(*timeout)
		if err != nil {
			logger.Error(err, "can't parse timeout")
			stopAndHelp()
		}

		go func() {
			// check if a wait behaviour is specified before starting the timeout.
			waitFor(*waitForStr, s, len(config.Nodes))

			time.Sleep(dur)
			e.Wait()

			// send interrupt to self for graceful shutdown
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}()
	}

	// check if mobility file is given, if so parse and run it
	var mob *emu.Mobility
	if len(config.Mobility.File) > 0 && !*noMobility {
		mobFile, err := os.Open(filepath.Join(configFolder, config.Mobility.File))
		if err != nil {
			panic(err)
		}

		commands, err := mobility.Parse(mobFile)
		if err != nil {
			panic(err)
		}

		mob = emu.NewMobility(e, commands).SetTickrate(config.Mobility.Tickrate).SetLoop(config.Mobility.Loop)

		if config.TimeScaling > 0 {
			if err := mob.SetTimeScaling(config.TimeScaling); err != nil {
				panic(err)
			}
		}

		// pass the mobility to the server
		s.SetMobility(mob)
	}

	s.SetDebug(*debug)
	s.SetLogger(logger)
	s.SetOrigin(config.Origin.X, config.Origin.Y)

	go func() {
		if err := s.Start(config.Web); err != nil {
			panic(err)
		}
	}()

	// TODO: really wait for the server to be started
	time.Sleep(time.Second)

	// check if commands should be executed for each node
	if len(config.Commands.PerNode) > 0 && !*skipCommands {
		specialFilter := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
		_ = os.Mkdir(filepath.Join(filepath.Dir(*logFile), "/cmdlogs/"), 0777)

		for i, node := range config.Nodes {
			params := map[string]interface{}{
				"i":         i,
				"nodeId":    node.ID,
				"bind":      config.Web,
				"nodeCount": len(config.Nodes),
			}

			for ci, cmd := range config.Commands.PerNode {
				logFile, err := os.OpenFile(filepath.Join(filepath.Dir(*logFile), "/cmdlogs/", fmt.Sprintf("%s-%d.txt", specialFilter.ReplaceAllString(node.ID, ""), ci)), os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
				if err != nil {
					panic(err)
				}

				executable, args, err := evalCommand(cmd, params)
				if err != nil {
					panic(err)
				}

				if strings.HasSuffix(executable, ".sh") {
					args = append(args, fmt.Sprint(i), node.ID, config.Web, fmt.Sprint(len(config.Nodes)))
				}

				logger.Info("running command", "node", node.ID, "cmd", executable, "args", strings.Join(args, " "))

				// write executed command to log file
				_, _ = logFile.WriteString(executable + " " + strings.Join(args, " ") + "\n\n")

				// start command
				cmd := exec.Command(executable, args...)
				cmd.Stdout = logFile
				cmd.Stderr = logFile
				cmd.Dir = configFolder
				if err := cmd.Start(); err != nil {
					panic(err)
				}

				runningCommands = append(runningCommands, RunningCommand{
					cmd:     cmd,
					logFile: logFile,
				})

				time.Sleep(time.Millisecond * time.Duration(config.Commands.DelayMs))
			}
		}
	}

	// run start commands
	if len(config.Commands.Start) > 0 && !*skipCommands {
		if config.Commands.BeforeStart > 0 {
			time.Sleep(time.Millisecond * time.Duration(config.Commands.BeforeStart))
		}

		waitFor(*waitForStr, s, len(config.Nodes))

		for i, cmd := range config.Commands.Start {
			logFile, err := os.OpenFile(filepath.Join(filepath.Dir(*logFile), "/cmdlogs/", fmt.Sprintf("start-cmd-%d.txt", i)), os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
			if err != nil {
				panic(err)
			}

			executable, args, err := evalCommand(cmd, map[string]interface{}{
				"nodeCount": len(config.Nodes),
			})
			if err != nil {
				panic(err)
			}

			logger.Info("running start command", "cmd", executable, "args", strings.Join(args, " "))

			// write executed command to log file
			_, _ = logFile.WriteString(executable + " " + strings.Join(args, " ") + "\n\n")

			// start command
			cmd := exec.Command(executable, args...)
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			cmd.Dir = configFolder
			if err := cmd.Start(); err != nil {
				panic(err)
			}

			runningCommands = append(runningCommands, RunningCommand{
				cmd:     cmd,
				logFile: logFile,
			})
		}
	}

	// start mobility at last
	if mob != nil {
		mob.Start()
	}

	// wait for commandline interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	// run end commands
	if !*skipCommands {
		for _, cmd := range config.Commands.End {
			executable, args, err := evalCommand(cmd, map[string]interface{}{
				"nodeCount": len(config.Nodes),
			})
			if err != nil {
				panic(err)
			}

			cmd := exec.Command(executable, args...)
			cmd.Dir = configFolder
			data, err := cmd.CombinedOutput()
			if err != nil {
				logger.Error(err, "end command error", "cmd", executable, "args", strings.Join(args, " "))
			} else {
				logger.Info("end command output", "cmd", executable, "args", strings.Join(args, " "))
			}
			if len(data) > 0 {
				fmt.Println("Out:\n" + string(data))
			}
		}
	}

	// kill all running child commands
	for _, rc := range runningCommands {
		logger.Info("killing command", "pid", rc.cmd.Process.Pid)
		if rc.logFile != nil {
			_ = rc.logFile.Sync()
			_ = rc.logFile.Close()
		}
		_ = rc.cmd.Process.Kill()
	}

	e.Wait()
}
