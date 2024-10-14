# filecoin-utils

This is a collection of utilities for working with the Filecoin blockchain.
- `address` displays multiple address formats including f0, f1, and eth.
- `chain` provides utilities for getting block and tipset information.
- `miner` provides utilities for getting miner information and sector expiration information.
    - `state` gets the miner state of a miner. You can know the penalty amount for actively terminating a sector after the sector is damaged. Refer to `MinerSectorsState` 
- `power` provides utilities for getting network power information.


## Build

```bash
    make build
```

## Run

### address

#### addrdescription

Prints the description of an address

Usage:
```bash
./bin/filecoin-utils utils addrdescription <address>
```

Example output:
```json
{
    "id": "f086971",
    "filecoin": "f1m2swr32yrlouzs7ijui3jttwgc6lxa5n5sookhi",
    "eth": "0x0000000000000000000000000000000000000000",
    "type": "unknown"
}
```

### chain

#### getblock

prints the block information of the given block hash

Usage:

```bash
# block_hash: block hash of the block to get
# flags:
#    --raw: print just the raw block header
./bin/filecoin-utils utils chain getblock <block_hash>
```

Example output:
```json
{ }
```

#### gettipset

prints the tipset information of the given block height

Usage:

```bash
# height: block height of the tipset to get
./bin/filecoin-utils utils chain gettipset <height>
```

Example output:
```json
{}
```

### miner

#### list
Lists all the miners in the network

Usage:

```bash
./bin/filecoin-utils utils miner list
```

Example output:

```json
[{}]
```

#### estimate-faulty

Estimate the amount of failure penalty that will occur after a given number of sectors are terminated

Usage:

```bash
# miner_id: miner id
# flags:
#    --pos, -p: Terminate start pos sectors
#    --number, -n: Terminate number sectors
./bin/filecoin-utils utils miner estimate-faulty <miner_id>
```

Example output:

```json
{}
```

#### state

Prints the state of a miner

Usage:
```bash
# miner_id: miner id
./bin/filecoin-utils utils miner state <miner_id>
```

Example output:
```json
{}
```

#### collectminer

Collect miner sector expiration information

Usage:
```bash
# miner_id: miner id
./bin/filecoin-utils utils miner collectminer <miner_id>
```

Example output:
```json
{}
```

#### collectsector

Collect all sector expiration information in the network

Usage:
```bash
./bin/filecoin-utils utils miner collectsector
```

Example output:
```json
{}
```

#### minersectors

Prints the sector information of a miner

Usage:
```bash
# miner_id: miner id
./bin/filecoin-utils utils miner minersectors <miner_id>
```

Example output:
```json
{}
```

### power

Prints the power information of the network

Usage:
```bash
./bin/filecoin-utils utils power
```

Example output:
```json
{}
```

