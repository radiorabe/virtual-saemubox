## Virtual Sämubox

Emulates a Sämubox with data queried from Pathfinder.

If you are not at RaBe and don't know exactly what this does you do not want to use this.

## Development

### Building

```bash
GOOS=linux GOARCH=amd64 /opt/local/bin/go build
```

### pre-commit hook

#### pre-commit configuration

```bash
pre-commit install
pre-commit install --hook-type commit-msg
pre-commit install-hooks
```

### Release Process

Create a git tag and push it to this repo or use the git web ui.

## License

This application is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, version 3 of the License.
