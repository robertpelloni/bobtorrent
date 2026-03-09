#!/usr/bin/env bash
cd "$(dirname "$0")/supernode-java"
# Run the specific CLI main class instead of the default DemoDashboard
./gradlew -q run --args="$*" -PmainClass=io.supernode.cli.NodeCLI
