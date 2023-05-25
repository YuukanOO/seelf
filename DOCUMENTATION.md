# Documentation

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

## Installation

Whatever installation you choose, you **MUST** have [Docker](https://docs.docker.com/get-docker/) installed to run seelf and a DNS correctly configured (with a wildcard redirecting to where seelf is hosted).

Compose and Git are packaged inside seelf itself so you don't have to bother with them.

The recommended way to deploy seelf is by using [Docker Compose](#with-docker-compose) since it makes the update process much more easier.

### With Docker

Don't forget to sets the [appropriate environment variables](#configuration) according to your needs.

```bash
docker run -d \
  -e "SEELF_ADMIN_EMAIL=admin@example.com" \
  -e "SEELF_ADMIN_PASSWORD=admin" \
  -e "BALANCER_DOMAIN=http://docker.localhost" \
  -v "/var/run/docker.sock:/var/run/docker.sock" \
  -v "seelfdata:/seelf/data" \
  -p "8080:8080" \
  yuukanoo/seelf
```

_Note: On windows, you may have to run the command on one line without the backslashes._

### With Docker Compose

Simply use the following `compose.yml` file and configure it according to your needs:

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf
    environment:
      - BALANCER_DOMAIN=http://docker.localhost # <- Change this to your own domain, applications will be deployed as subdomains
      - SEELF_ADMIN_EMAIL=admin@example.com # <- Change this
      - SEELF_ADMIN_PASSWORD=admin # <- Change this
      # - HTTP_SECURE= # By default, the seelf server use Secure cookies if BALANCER_DOMAIN starts with https://, you can override this behavior by setting it explicitly to true or false here. If you wish to go back to the initial value, just set HTTP_SECURE= as demonstrated here
      # - DEPLOYMENT_DIR_TEMPLATE={{ .Number }}-{{ .Environment }} # You can configure the deployment build directory path if you want to keep every deployment source files for example.
      # - ACME_EMAIL=youremail@provider.com # <- If BALANCER_DOMAIN starts with https://, let's encrypt certificate will be used and the email associated will default to SEELF_ADMIN_EMAIL but you can override it if you need to
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - seelfdata:/seelf/data # The /seelf/data directory contains the database, configuration file and everything deployed by seelf, so keep it :)

volumes:
  seelfdata:
```

_Note: If you want to build the image yourself (because your platform is not supported for example), you can use the command `docker build -t yuukanoo/seelf .` or use the `compose.yml` in this repository which build the image._

### From sources

You'll need **Go** installed on your machine (at least 1.20) and **Node** for the frontend stuff (at least 1.18).

_Note: on **Windows**, you will need a gcc compiler such as [tdm gcc](https://jmeubank.github.io/tdm-gcc/) to build the sqlite3 driver correctly._

Retrieve the seelf sources by either cloning the repository or downloading them on the **Releases** page and run:

```bash
make build
./seelf serve
```

## Exposing seelf itself

You probably want to expose seelf itself on an url without a port and with a valid certificate. You can manage this part yourself or just leverage the traefik proxy deployed by seelf.

To do this, we recommend to use the [docker compose installation](#with-docker-compose) and add the missing parts (see comments).

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf
    environment:
      - BALANCER_DOMAIN=http://docker.localhost
      - SEELF_ADMIN_EMAIL=admin@example.com
      - SEELF_ADMIN_PASSWORD=admin
      - HTTP_SECURE= # Force fallback to the default handling of http secure (based on the BALANCER_DOMAIN)

    # Remove the ports part since they are not needed anymore
    labels:
      - traefik.enable=true # Enable traefik for seelf
      - traefik.http.routers.seelf.rule=Host(`seelf.docker.localhost`) # Expose seelf on the given host. Replace seelf.docker.localhost by where you want seelf to be available
      # - traefik.http.routers.seelf.tls.certresolver=seelfresolver # If BALANCER_DOMAIN starts with https://, uncomment this line too to generate needed certificates
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - seelfdata:/seelf/data

volumes:
  seelfdata:

networks:
  default:
    name: seelf-public # Connect to the public network used by traefik to make seelf available (since ports have been removed)
```

_Note: You can also do this directly with the docker command with `-l "traefik.enable=true"` and so on, `docker network connect seelf-public <seelf container id>` but that's way more complicated._

## Updating

### With Docker

Since the configuration has been saved in the volume `seelfdata`, you can omit the settings used when launching seelf for the first time.

```bash
docker pull yuukanoo/seelf && docker rm $(docker stop $(docker ps -a -q --filter="ancestor=yuukanoo/seelf")) && docker run -d -v "/var/run/docker.sock:/var/run/docker.sock" -v "seelfdata:/seelf/data" -p "8080:8080" yuukanoo/seelf
```

### With Docker Compose

Go where the initial `compose.yml` file has been created and run:

```bash
docker compose pull && docker compose up -d
```

### From sources

Simply build the application again with the latest sources and you're good to go.

## Configuration

seelf can be configured by a yaml configuration file (as below) and/or environment variables (see comments for the appropriate name). Those environment variables can also be set using a `.env` or `.env.local` in the working directory when launching seelf. Either way, the resulting configuration will be written to the application folder during the application startup so the next run may not include initial configuration options.

When configured using a file, you can provide it with the `-c <your_file.yml>` or `--config <your_file.yml>`. Here is the full configuration file:

```yaml
verbose: true # SEELF_DEBUG Verbose mode
data:
  path: "~/.config/seelf" # DATA_PATH Where data produced by seelf will be saved (apps deployment, logs, sqlite db, ...)
  deployment_dir_template: "{{ .Environment }}" # DEPLOYMENT_DIR_TEMPLATE Go template, determine the directory where the build will occur (use '{{ .Number }}-{{ .Environment }}' if you want to keep all application deployment sources)
http:
  host: 0.0.0.0 # HTTP_HOST Host to listen to
  port: 8080 # HTTP_PORT,PORT Port to listen to
  secure: false # HTTP_SECURE Wether or not the web server is served over https. If omitted, determine this information from the BALANCER_DOMAIN. It controls wether or not cookie are sent with the Secure flag and the scheme used on the Location header of created resources
  secret: "<generated if empty>" # HTTP_SECRET Secret key to use when signing cookies
balancer:
  domain: "http://docker.locahost" # BALANCER_DOMAIN Main domain to use when deploying an application. If starting with https://, Let's Encrypt certificates will be generated and the acme email is mandatory. If you change this domain afterward, you'll have to redeploy your apps for now
  acme:
    email: # ACME_EMAIL Email used by let's encrypt, if no one is provided, it will use the SEELF_ADMIN_EMAIL if possible, else the application won't launch
```

Additionaly, you **MUST** provide a valid `SEELF_ADMIN_EMAIL` and `SEELF_ADMIN_PASSWORD` during the first startup to create the admin account. Those values **MUST** be provided with environment variables and will be used only if no user account exist yet.

## Usage

Deploying an application using seelf is really straightforward:

- **Sign in** to the user dashboard where you have deployed seelf
- **Register** an application
  - (optional) Connect it to a git repository
  - (optional) Define environment variables per environment / per service
- **Request** a deployment
- **Profit!**

### Environments

For now, seelf supports two environments: **production** and **staging**. In the future, arbitrary environment may be allowed.

The production environment has a special meaning when dealing with which subdomain will be given to your services (see [notes below](#how-are-services-exposed)).

A [compose profile](https://docs.docker.com/compose/profiles/) with the same name as the environment will also be applied when deploying your application and can help you prevent some services to be activated based on the environment.

### Deployment types

You can deploy an application using either:

- A raw **compose file**
- A **source archive (.tar.gz)**
- A **git branch / commit** (if you have configured a git repository when registering your application)

Either way, you'll need to have a valid compose file at the root of your project.

### How are services exposed?

Let's take an example for an application registered with the name `sandbox` and a `production` deployment targeting a balancer domain sets to the default value `http://docker.localhost`.

Whatever method you choose when deploying your application, seelf will look for a compose file at the root of a deployment build directory in this order:

```
- compose.seelf.production.yml
- compose.seelf.production.yaml
- docker-compose.seelf.production.yml
- docker-compose.seelf.production.yaml
- compose.production.yml
- compose.production.yaml
- docker-compose.production.yml
- docker-compose.production.yaml
- compose.seelf.yml
- compose.seelf.yaml
- docker-compose.seelf.yml
- docker-compose.seelf.yaml
- compose.yml
- compose.yaml
- docker-compose.yml
- docker-compose.yaml
```

Once it has found a valid compose file, it will apply some heuristics to determine which services should be exposed and where.

Let's say you have this `compose.yml` file:

```yml
services:
  app:
    restart: unless-stopped
    build: .
    environment:
      - DSN=postgres://app:apppa55word@db/app?sslmode=disable
    depends_on:
      - db
    ports:
      - "8080:8080"
  sidecar:
    image: traefik/whoami
    ports:
      - "8889:80"
    profiles:
      - production
  stagingonly:
    image: traefik/whoami
    ports:
      - "8888:80"
    profiles:
      - staging
  db:
    restart: unless-stopped
    image: postgres:14-alpine
    volumes:
      - dbdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=app
      - POSTGRES_PASSWORD=apppa55word
volumes:
  dbdata:
```

When deploying this project on seelf, it will:

- Build an image for the `app` service named `sandbox/app:production`
- Expose the `app` service on the default subdomain `http://sandbox.docker.localhost` because that's the first service in **alphabetical order** which has **port mappings defined**. If environment variables has been defined for the `app` service in the production environment, they will overwrite what's in the compose file
- expose the `sidecar` service on `http://sidecar.sandbox.docker.localhost` because it has port mappings too and the production profile is activated
- skip the `stagingonly` service because we have requested a production deployment
- run the `db` service without exposing it because it does not have port mappings defined and has such will be kept private and use any environment variables defined for the `db` service in the production environment.

To expose those services, seelf will add the needed traefik labels based on the balancer domain you've set when configuring it. If your domain starts with `https://`, Let's encrypt certificates will be generated for you.
