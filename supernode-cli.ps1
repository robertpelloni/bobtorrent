Set-Location -Path "$PSScriptRoot\supernode-java"
.\gradlew.bat -q run --args="$args" -PmainClass="io.supernode.cli.NodeCLI"
