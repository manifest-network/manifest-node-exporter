# Collectors

The `collectors` package contains various metric collectors for the Manifest Node Exporter. Each collector is responsible
for gathering specific metrics from the Manifest node and exposing them via Prometheus format.

## Overview

Collectors are the core components that gather metrics from different aspects of a Manifest node. They implement the
Prometheus collector interface and provide customized metric collection logic for specific node characteristics.

