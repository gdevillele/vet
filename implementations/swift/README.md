# Swift Implementation

This directory will contain the Swift-native `vet` runner.

Expected usage:

```sh
swift run vet path/to/project
```

The Swift implementation should consume the shared rule contract in `../../spec`
and emit the same diagnostic shape as the other implementations.

Likely parser option:

- SwiftSyntax, with SwiftPM integration once the CLI shape is stable.
