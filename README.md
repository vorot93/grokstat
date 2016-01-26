# GrokStat
Retrieves information about game servers. Inspired by [QStat](https://github.com/multiplay/qstat), written in Go. Yet even more simple, extensible and fast.

GrokStat accepts input data as JSON via stdin. The result is displayed in JSON form as well. In order to run a simple query you need to specify protocol and array of hosts to check. Please refer to the example below for more information.

The server query is asynchronous, done via inbuilt UDP server.

## Protocols
M stands for master server support. S stands for individual game server query support.

### Supported
- **M** **S** | Quake-derived games:
 - Quake II
 - Quake III
 - Xonotic
 - OpenArena
 - Warsow
 - Unvanquished
 - Soldiers of Fortune 2
- **M** **S** | OpenTTD
- **M** **S** | Teeworlds
- **M** **S** | Steam / SourceQuery

### Planned
- TeamSpeak 3
- Mumble

## Get it
### Dependencies
	go get -u github.com/BurntSushi/toml github.com/jteeuwen/go-bindata/...
### GrokStat itself
	git clone https://github.com/grokstat/grokstat.git
    cd grokstat && make build
    bin/grokstat

## Example
    echo '{"protocol": "q3m", "hosts": ["master3.idsoftware.com"]}' | bin/grokstat

## License
This program is free software; you can redistribute it and/or modify it under the terms of the GNU General Public License version 3, as published by the Free Software Foundation.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
