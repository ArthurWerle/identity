# Swagger Documentation

This directory contains auto-generated Swagger/OpenAPI documentation.

## Generating Documentation

To generate the Swagger documentation, run:

```bash
make swagger
```

Or directly with swag:

```bash
swag init -g cmd/server/main.go -o docs
```

This will generate the following files:
- `docs.go` - Go code with embedded documentation
- `swagger.json` - OpenAPI JSON specification
- `swagger.yaml` - OpenAPI YAML specification

## Viewing Documentation

Once the service is running, you can access the Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

## Prerequisites

Install swag CLI:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

## Note

The generated files are listed in `.gitignore` and should not be committed to version control.
They will be generated as part of the build process.
