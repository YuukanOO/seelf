# Installation

Whatever installation method you choose, you **MUST** have:

- [Docker >= v18.0.9](https://docs.docker.com/get-docker/) installed (2019-09-03)
- A DNS correctly configured with a wildcard redirecting to the [target](/reference/targets)

**Compose** and **Git** are packaged inside seelf itself so you don't have to bother with them.

## With Compose (recommended) {#with-compose}

Simply save the following `compose.yml` file in a folder and [configure it according to your needs](/guide/configuration):

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf:latest # <- You probably better sets the version explicitly here
    environment:
      - ADMIN_EMAIL=admin@example.com # <- Change this
      - ADMIN_PASSWORD=admin # <- Change this
      # (...) sets other environment variables here, see the configuration documentation
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ssh:/root/.ssh # If you deploy on remote servers, keep ssh related configurations
      - data:/seelf/data # The /seelf/data directory contains the database, configuration file and everything deployed by seelf, so keep it :)

volumes:
  ssh:
  data:
```

## With Docker

Don't forget to sets the [appropriate environment variables](/guide/configuration) according to your needs.

```sh
docker run -d \
  -e "ADMIN_EMAIL=admin@example.com" \
  -e "ADMIN_PASSWORD=admin" \
  -v "/var/run/docker.sock:/var/run/docker.sock" \
  -v "seelfdata:/seelf/data" \
  -v "seelfssh:/root/.ssh" \
  -p "8080:8080" \
  yuukanoo/seelf
```

::: info
On windows, you may have to run the command on one line without the backslashes.
:::

## From sources

You'll need **Go** installed on your machine (at least 1.21) and **Node** for the frontend stuff (at least 1.18).

::: info
On **Windows**, you will need a gcc compiler such as [tdm gcc](https://jmeubank.github.io/tdm-gcc/) to build the sqlite3 driver correctly.
:::

Retrieve the seelf sources by either [cloning the repository](https://github.com/YuukanOO/seelf) or downloading them on the [Releases page](https://github.com/YuukanOO/seelf/releases) and run:

```sh
make build && ./seelf serve
```

## Exposing seelf itself {#exposing-seelf}

You probably want to expose seelf itself on an url without a port and with a valid certificate. You can manage this part yourself or just leverage the proxy deployed by seelf.

To do this, we recommend to use the [docker compose installation](#with-compose) and add the missing parts.

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf:latest # <- You probably better sets the version explicitly here
    container_name: seelf # Should match the user portion of EXPOSED_ON when exposing seelf using a local target // [!code ++]
    environment:
      - ADMIN_EMAIL=admin@example.com # <- Change this
      - ADMIN_PASSWORD=admin # <- Change this
      # (...) sets other environment variables here, see the configuration documentation
      - EXPOSED_ON=http://seelf@docker.localhost # <- Change this to where you want seelf to be exposed, the user part represents the container name, see above. // [!code ++]
      - HTTP_SECURE= # Force fallback to the default handling of http secure (based on EXPOSED_ON if set) // [!code ++]
    ports: // [!code --]
      - "8080:8080" // [!code --]
    labels: // [!code ++]
      - app.seelf.exposed=true // [!code ++]
      - app.seelf.subdomain=myseelf # Subdomain where seelf will be exposed on the default target represented by EXPOSED_ON, here http://myseelf.docker.localhost // [!code ++]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ssh:/root/.ssh # If you deploy on remote servers, keep ssh related configurations
      - data:/seelf/data # The /seelf/data directory contains the database, configuration file and everything deployed by seelf, so keep it :)

volumes:
  ssh:
  data:
```

::: info
If set, the `EXPOSED_ON` will be used to determine:

- The name of the seelf container itself which must be attached to the local target,
- The default local target URL if no one exists yet.

If a local target already exists, the container will be attached to it without updating the target URL.
:::
