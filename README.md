#  StreamingFast Solana Command Line Tool
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/solana-go)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A command-line client to Solana clusters - by StreamingFast (streamingfast.io)

## Installation

1. Install from https://github.com/streamingfast/slnc/releases

**or**

2. Install using Homebrew on macOS
```bash
brew install streamingfast/tap/slnc
```
``
3. Build from source with (Golang 1.17 required):

```bash
go install github.com/streamingfast/slnc/cmd/slnc
```

## Usage

```bash
slnc get balance EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v
1461600 lamports

slnc get account EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v
{
  "lamports": 1461600,
  "data": [
    "AQAAABzjWe1aAS4E+hQrnHUaHF6Hz9CgFhuchf/TG3jN/Nj2gCa3xLwWAAAGAQEAAAAqnl7btTwEZ5CY/3sSZRcUQ0/AjFYqmjuGEQXmctQicw==",
    "base64"
  ],
  "owner": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
  "executable": false,
  "rentEpoch": 108
}

slnc spl get mint SRMuApVNdxXokk5GT7XD5cUUgXMBCoAz2LHeuAoKWRt
Supply               9999647664800000
Decimals             6
No mint authority
No freeze authority

slnc serum list markets
...
ALEPH/USDT   EmCzMQfXMgNHcnRoFwAdPe1i2SuiSzMj1mx6wu3KN2uA
ALEPH/WUSDC  B37pZmwrwXHjpgvd9hHDAx1yeDsNevTnbbrN9W12BoGK
BTC/USDT     8AcVjMG2LTbpkjNoyq8RwysokqZunkjy3d5JDzxC6BJa
BTC/WUSDC    CAgAeMD7quTdnr6RPa7JySQpjf3irAmefYNdTb6anemq
ETH/USDT     HfCZdJ1wfsWKfYP2qyWdXTT5PWAGWFctzFjLH48U1Hsd
ETH/WUSDC    ASKiV944nKg1W9vsf7hf3fTsjawK6DwLwrnB2LH9n61c
SOL/USDT     8mDuvJJSgoodovMRYArtVVYBbixWYdGzR47GPrRT65YJ
SOL/WUSDC    Cdp72gDcYMCLLk3aDkPxjeiirKoFqK38ECm8Ywvk94Wi
...

slnc serum get market 7JCG9TsCx3AErSV3pvhxiW4AbkKRcJ6ZAveRmJwrgQ16
Price    Quantity  Depth
Asks
...
527.06   444.09    ####################
393.314  443.52    ###############
463.158  443.17    ###########
200      442.63    ######
234.503  442.54    ####
50       441.86    ##
61.563   441.47    #
84.377   440.98
-------  --------
10       439.96
193.303  439.24    ##
50       438.94    ##
0.5      438.87    ##
247.891  437.65    #####
458.296  436.99    #########
452.693  435.68    ##############
372.722  435.12    ##################
0.043    431.94    ##################
...
```

## Release

Use the `./bin/release.sh` Bash script to perform a new release. It will ask you questions
as well as driving all the required commands, performing the necessary operation automatically.
The Bash script runs in dry-mode by default, so you can check first that everything is all right.

Releases are performed using [goreleaser](https://goreleaser.com/).

## Contributing

**Issues and PR in this repo related strictly to the solana go library.**

Report any protocol-specific issues in their
[respective repositories](https://github.com/streamingfast/streamingfast#protocols)

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.

This codebase uses unit tests extensively, please write and run tests.

## License

[Apache 2.0](LICENSE)
