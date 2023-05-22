# seelf : Painless self-hosted deployment platform

I mean, for real!

https://github.com/YuukanOO/seelf/assets/939842/8c307439-8970-4c74-9409-47995a8771bc

_The seelf initial public version has some limitation (only local Docker engine, single user mostly) but you can check out [the roadmap](https://github.com/YuukanOO/seelf/milestone/1) to see what's planned!_

## Goal

[seelf](https://github.com/YuukanOO/seelf) is a self-hosted deployment platform to make deploying an application stack as easy as possible.

This project was born because, as a developer, I often have tiny toy applications to deploy and found it somehow frustating. I've tested many self-hosted PaaS such as Dokku and Caprover but none fit me.

The initial idea that led to this project was to take a developer **docker compose** file representing a project stack (as seen in many projects nowadays) and use it **without any modification** to deploy it on your own infrastructure.

Key aspects of seelf are:

- Tiny (in size)
- Lightweight (in resource usage)
- Reliable
- Easy to understand

_Althought Docker is the only backend supported at the moment, I would like to investigate to enable other ones too. Remote Docker or Podman for example._

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- A DNS correctly configured (with a wildcard redirecting to where seelf is hosted)

## Quick start

Want to give seelf a try locally? Run those commands to start it (the network here represents the public gateway used by the balancer internally so you have to name it `seelf-public`):

```bash
docker network create seelf-public
docker run -d \
  --name seelf \
  -e "SEELF_ADMIN_EMAIL=admin@example.com" \
  -e "SEELF_ADMIN_PASSWORD=admin" \
  -l "traefik.enable=true" \
  -l "traefik.docker.network=seelf-public" \
  -l "traefik.http.routers.seelf.rule=Host(\`seelf.docker.localhost\`)" \
  -v "/var/run/docker.sock:/var/run/docker.sock" \
  -v "seelfdata:/seelf/data" \
  --network seelf-public \
  --restart=unless-stopped \
  yuukanoo/seelf
```

_Note: Traefik will be deployed by seelf itself when starting up so you don't have to worry about it._

Head over [http://seelf.docker.localhost](http://seelf.docker.localhost) and sign in using `admin@example.com` and `admin` as password!

To quickly check how seelf behaves, you can use [examples](examples/README.md) packaged as `.tar.gz` archives in this repository.

See all [available options in the documentation](DOCUMENTATION.md#configuration) to get more serious and configure seelf for you server.

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
