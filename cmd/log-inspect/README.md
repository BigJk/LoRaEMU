# LoRaEMU Log Inspector

```
  _        ___      ___ __  __ _   _ 
 | |   ___| _ \__ _| __|  \/  | | | |
 | |__/ _ \   / _` | _|| |\/| | |_| |
 |____\___/_|_\__,_|___|_|  |_|\___/
    - LogInspect
------------------------------------------
Daniel Schmidt <info@daniel-schmidt.me>

:: Log Inspector is a helper Utility for the LoRaEMU to run expressions on trace logs.

USAGE:
  -expr string
        the expression that should be evaluated
  -input string
        specify path to LoRaEMU trace log.
  -output string
        the operation that should be done on the found entries (e.g. print, count) (default "print")
```

The LoRaEMU Log Inspector makes it possible to run custom expressions over a trace log that was written by the LoRaEMU. This can potentially help to quickly analyse certain metrics based on the result of an emulation session and might be useful in automated tests.

With that it's easy to:
- Count the number of collisions
- Count the number of collisions on a certain node
- Count the number of received packets
- Get all logs corresponding to a certain node

## Expression

The expression specified by ``-expr`` will be run on each line of the trace log. Each line of the trace log represents a JSON object of the following kind:

```json
{"time":"2022-12-05T21:21:49.219622+01:00","event":"NodeUpdated","nodeId":"Node2","data":null}
```

The evaluation is done by the [govaluate](https://github.com/Knetic/govaluate) library, so check the documentation for further information on the syntax and available operators.

## Output Types

The output type is specified by the ``-type`` parameter.

- ``print``: Will print all JSON encoded log entries that match the expression.
- ``count``: Counts how many log entries match the expression.
- ``sum``: Sums a integer / float value.


## Examples

- Get logs after a certain time: ``-expr "time > '2022-12-02T07:53:00.808676+01:00'" -output print``
- Get logs for a node: ``-expr "nodeId == 'Node1'" -output print``
- Count collisions: ``-expr "type == 'NodeCollision'" -output count``
- Count collisions on a node: ``-expr "type == 'NodeCollision' && nodeId == 'Node1'" -output count``
- Count sends done: ``-expr "type == 'NodeSending'" -output count``
- Count sends on a node: ``-expr "type == 'NodeSending' && nodeId == 'Node1'" -output count``
- Count received packets: ``-expr "type == 'NodeReceived'" -output count``
- Count received packets on a node: ``-expr "type == 'NodeReceived' && nodeId == 'Node1'" -output count``
- Sum airtime ``-expr "event == 'NodeSending' ? data_airtime : 0.0" -output sum``