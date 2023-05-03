package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/fatih/structs"
	"github.com/nqd/flat"
)

func init() {
	structs.DefaultTagName = "json"
}

func printHelp() {
	fmt.Println("  _        ___      ___ __  __ _   _ \n | |   ___| _ \\__ _| __|  \\/  | | | |\n | |__/ _ \\   / _` | _|| |\\/| | |_| |\n |____\\___/_|_\\__,_|___|_|  |_|\\___/\n    - LogInspect")
	fmt.Println("------------------------------------------")
	fmt.Printf("Daniel Schmidt <info@daniel-schmidt.me>\n\n")
	fmt.Printf(":: Log Inspector is a helper Utility for the LoRaEMU to run expressions on trace logs.\n\nUSAGE:\n")
	flag.PrintDefaults()
}

func main() {
	input := flag.String("input", "", "specify path to LoRaEMU trace log.")
	exprStr := flag.String("expr", "", "the expression that should be evaluated")
	opType := flag.String("output", "print", "the operation that should be done on the found entries (e.g. print, count, sum)")
	flag.Parse()

	if len(*input) == 0 {
		fmt.Printf("Error: no input given\n\n")
		printHelp()
		return
	}

	if len(*exprStr) == 0 {
		fmt.Printf("Error: no input given\n\n")
		printHelp()
		return
	}

	// open the file
	file, err := os.Open(*input)
	if err != nil {
		fmt.Printf("Error: can't open file (%s)\n\n", err)
		printHelp()
		return
	}

	// create the evaluator
	expr, err := govaluate.NewEvaluableExpression(*exprStr)
	if err != nil {
		fmt.Printf("Error: can't parse expression (%s)\n\n", err)
		printHelp()
		return
	}

	var found []emu.LogEntry
	var sum float64
	var concat []string

	scanner := bufio.NewScanner(file)

scannerFor:
	for scanner.Scan() {
		var entry emu.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			panic(err)
		}

		// convert struct to map[string]interface{} so that the evaluator can work with it
		entryMap := structs.Map(entry)
		entryMap["event"] = string(entryMap["event"].(emu.Event)) // evaluator can't compare String to emu.Event, so we cast it to string beforehand

		// convert the time to a unix timestamp so that the evaluator can use operations on it
		if entryTime, ok := entryMap["time"].(time.Time); ok {
			entryMap["time"] = entryTime.Unix()
		}

		entryMap, err = flat.Flatten(entryMap, &flat.Options{
			Delimiter: "_",
		})
		if err != nil {
			panic("could not flat log entry")
		}

		// evaluate expression
		res, err := expr.Evaluate(entryMap)
		if err != nil {
			panic(err)
		}

		switch *opType {
		case "sum":
			val, validType := res.(float64)
			if !validType {
				panic("expression didn't return a sumable type")
			}

			sum += val
			continue scannerFor
		case "concat":
			str := fmt.Sprint(res)
			if len(str) > 0 {
				concat = append(concat, str)
			}
			continue scannerFor
		}

		ok, validType := res.(bool)
		if !validType {
			panic("expression didn't return bool")
		}

		if ok {
			found = append(found, entry)
		}
	}

	// run the operation type on the found entries
	switch *opType {
	case "print":
		for _, e := range found {
			m, err := json.Marshal(e)
			if err != nil {
				panic(err)
			}

			fmt.Println(string(m))
		}
	case "count":
		fmt.Println(len(found))
	case "sum":
		fmt.Println(sum)
	case "concat":
		fmt.Println(strings.Join(concat, ";"))
	}
}
