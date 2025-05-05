# Manifest Node Exporter

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/liftedinit/manifest-node-exporter/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/liftedinit/manifest-node-exporter/tree/main)

A Prometheus exporter for collecting metrics from Manifest blockchain nodes.

## Installation

Download the latest release from the [releases page](https://github.com/liftedinit/manifest-node-exporter/releases)

## Quick Start

```bash
manifest-node-exporter serve grpc-address [flags]
```

where `grpc-address` is the address of the `manifest-ledger` gRPC server.

The exporter will start a Prometheus metrics server on `0.0.0.0:2112` by default.

## Global Flags

| Flag                | Description                                                                               |
|---------------------|-------------------------------------------------------------------------------------------|
| `-h`, `--help`      | help for manifest-node-exporter                                                           |
| `-l`, `--logLevel` | Set the log level. Available levels: `debug`, `info`, `warn`, `error`. Default is `info`. |

## Serve Flags

| Flag                | Description                                                                               |
|---------------------|-------------------------------------------------------------------------------------------|
| `-h`, `--help`      | help for serve                                                                           |
| `--insecure` | Skip TLS verification for gRPC connection. Default is `false`.                           |
| `--listen-address` | Address to listen on for Prometheus metrics. Default is `0.0.0.0:2112`.                |

## Metrics

| Metric Name                         | Description                                                               |
|-------------------------------------|---------------------------------------------------------------------------|
| `manifest_tokenomics_denom_info`    | Information about the token denominations (symbol, denom, name, display). |  
| `manifest_tokenomics_total_supply`  | Total supply for a given token.                                           |
| `manifest_tokenomics_token_count`   | The number of different tokens hosted on the Manifest blockchain. |
| `manifest_tokenomics_denom_grpc_up` | Whether the gRPC query for the token denomination was successful. |
| `manifest_tokenomics_count_grpc_up` | Whether the gRPC query for the token count was successful. |