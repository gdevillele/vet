# Swift Implementation

This directory contains the Swift-native `vet` runner.

Usage:

```sh
swift run vet path/to/project
```

With config:

```sh
swift run vet --config ../../spec/config/v1.example.yaml path/to/project
swift run vet -c ../../spec/config/v1.example.yaml path/to/project
```

Supported rules:

- `VET001`: maximum function parameters;
- `VET002`: required source file header;
- `VET003`: minimum source file header length;
- `VET004`: maximum source file header length.

The first implementation uses a lightweight lexical analyzer. The parser
boundary is isolated so it can be replaced with SwiftSyntax later without
changing the CLI contract.
