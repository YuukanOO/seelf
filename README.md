# seelf : Painless self-hosted deployment platform

I mean, for real!

https://github.com/YuukanOO/seelf/assets/939842/d234bf40-1927-4057-a62b-8357c935506b

_The seelf initial public version has some limitation (only local Docker engine, single user mostly) but you can check out [the roadmap](https://github.com/YuukanOO/seelf/milestone/1) to see what's planned!_

## Goal

Got an already working docker compose file for your project ? Just send it to your [seelf](https://github.com/YuukanOO/seelf) instance and _boom_, that's live on your own infrastructure with all services correctly deployed and exposed on nice urls as needed! See [the documentation](DOCUMENTATION.md) for more information.

_Althought Docker is the only backend supported at the moment, I would like to investigate to enable other ones too. Remote Docker or Podman for example._

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- A DNS correctly configured (with a wildcard redirecting to where seelf is hosted)

## Quick start

Want to give seelf a try?

```bash
docker run -d -e "SEELF_ADMIN_EMAIL=admin@example.com" -e "SEELF_ADMIN_PASSWORD=admin" -v "/var/run/docker.sock:/var/run/docker.sock" -v "seelfdata:/seelf/data" -p "8080:8080" yuukanoo/seelf
```

Head over [http://localhost:8080](http://localhost:8080) and sign in using `admin@example.com` and `admin` as password!

To quickly check how seelf behaves, you can use [examples](examples/README.md) packaged as `.tar.gz` archives in this repository.

See all [available options in the documentation](DOCUMENTATION.md#installation) to get more serious and configure seelf for your server.

## Documentation

See the [documentation](DOCUMENTATION.md) in this repository. If you need more help, feel free to open an issue!

## Contributing

Oh nice! Let's build together!

The `Makefile` contains some target such as:

- `make serve-front`: run the dashboard UI for development
- `make serve-back`: serve the backend API
- `make test`: launch all tests
- `make ts`: print the current timestamp (mostly used when creating migrations)
- `make build`: build seelf for the current platform

_Note: on **Windows**, you will need a gcc compiler such as [tdm gcc](https://jmeubank.github.io/tdm-gcc/) to build the sqlite3 driver correctly._

### Architecture

See the [architecture overview](ARCHITECTURE.md).
