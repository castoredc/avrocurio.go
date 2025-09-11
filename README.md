# AvroCurio

[Apache Avro] serialization/deserialization with [Confluent Schema Registry framing] using [Apicurio Schema Registry].

This is a Go port of the [Python AvroCurio library], designed to provide the same functionality with idiomatic Go patterns.

## Quickstart

Refer to [ExampleAvroSerializer] which shows a basic usage example of how to use this library.
Don't forget to add it your project first:

```bash
go get github.com/castoredc/avrocurio.go
```

## Confluent Schema Registry Wire Format

AvroCurio implements the Confluent Schema Registry wire format:

```
+----------------+------------------+------------------+
| Magic Byte     | Schema ID        | Avro Payload     |
| (1 byte = 0x0) | (4 bytes, BE)    | (remaining)      |
+----------------+------------------+------------------+
```

- **Magic Byte**: Always `0x0` to identify Confluent wire format
- **Schema ID**: 4-byte big-endian integer referencing the schema in the registry
- **Avro Payload**: Standard Avro binary-encoded data

### Schema Caching

Schema caching is handled automatically by the ApicurioClient for performance.

Successful schema lookups are cached indefinitely.
Failed lookups are cached with a TTL to avoid constant repeated failed requests, but allow retries over time in case the error is transient.

## Development

### Requirements

Integration tests require a running Apicurio Registry.
Running it through Docker using [Compose][docker-compose] is easiest:

```bash
docker compose up
```

Port 8080 is assumed by default, but you can set `APICURIO_URL` to point to a different instance.

### Running Tests

```bash
# Run unit tests
make test

# Run tests with race detection
make test-race

# Run tests with coverage
make test-cover

# Run integration tests (requires running Apicurio Registry)
make test-integration
```

## License

AvroCurio is open-source software released under the [BSD-2-Clause Plus Patent License].
This license is designed to provide: a) a simple permissive license; b) that is compatible with the GNU General Public License (GPL), version 2; and c) which also has an express patent grant included.

Please review the [LICENSE] file for the full text of the license.

[Apache Avro]: https://avro.apache.org/
[Apicurio Schema Registry]: https://www.apicur.io/registry/
[BSD-2-Clause Plus Patent License]: https://spdx.org/licenses/BSD-2-Clause-Patent.html
[Confluent Schema Registry framing]: https://docs.confluent.io/platform/current/schema-registry/fundamentals/serdes-develop/index.html#wire-format
[ExampleAvroSerializer]: example_test.go
[LICENSE]: LICENSE
[Python AvroCurio library]: https://github.com/castoredc/avrocurio
[docker-compose]: https://docs.docker.com/compose/
