# Targets

Targets represents an **host** where your deployments will be exposed. When configuring an application, you choose on which target a specific environment should be deployed.

::: info
For now, only one target per host is allowed.
:::

## Providers {#providers}

You must choose one provider kind when creating a target. Some providers have specific parameters for you to configure how things work.

::: warning
Whatever provider you choose, you should make sure your **DNS is correctly configured with a wildcard redirecting to the target host**, the [DigitalOcean procedure](https://docs.digitalocean.com/glossary/wildcard-record/) can be applied to your specific provider.
:::

### Docker

Uses [Docker Compose](https://docs.docker.com/compose/) to launch your services by looking in the project root for specific files.

::: warning
[Docker >= (v18.0.9) must be installed](https://docs.docker.com/get-docker/) on the target!

Docker engine `v1.41` can sometimes cause issues when attaching multiple networks on container creation. If you have any issue, consider updating.
:::

#### Files looked at

When trying to process a single [deployment](/reference/deployments), it will try to find a compose file in the following order at the **project root**:

```
compose.seelf.ENVIRONMENT.yml
compose.seelf.ENVIRONMENT.yaml
docker-compose.seelf.ENVIRONMENT.yml
docker-compose.seelf.ENVIRONMENT.yaml
compose.ENVIRONMENT.yml
compose.ENVIRONMENT.yaml
docker-compose.ENVIRONMENT.yml
docker-compose.ENVIRONMENT.yaml
compose.seelf.yml
compose.seelf.yaml
docker-compose.seelf.yml
docker-compose.seelf.yaml
compose.yml
compose.yaml
docker-compose.yml
docker-compose.yaml
```

Where `ENVIRONMENT` will be one of `production`, `staging`.

#### Exposing services

Once a valid compose file has been found, **seelf** will apply some **heuristics** to determine which services should be exposed and where.

It will consider any service with **port mappings** to be exposed.

::: info Why relying on **ports mappings**?
When working on a local compose stack, you make **services available by defining ports mappings**. By using this **heuristic**, we make the transition from local to remote a breeze.
:::

The first service in **alphabetical order** will take the [default application subdomain](/reference/applications#environments). Every other services exposed will be on a subdomain of that default one.

::: warning
For now, only **one port exposed per service** is allowed and **only HTTP** services can be exposed. Exposing TCP or UDP services is on [the roadmap](https://github.com/YuukanOO/seelf/issues/17).
:::

#### Labels appended by seelf

To identify which resources are managed by seelf, some **docker labels** are appended during the deployment process. Some labels such as `app.seelf.application`, `app.seelf.target` and `app.seelf.environment` are appended to each resources: container, networks, volumes and images built while the others labels are only appended to the container.

| Name                  | Description                                                                            |
| --------------------- | -------------------------------------------------------------------------------------- |
| app.seelf.exposed     | Only used to identify the seelf container when exposing it through a local target      |
| app.seelf.application | ID of the application                                                                  |
| app.seelf.environment | [Environment](/reference/applications#environments) of the resource                    |
| app.seelf.target      | ID of the target on which the container must be exposed                                |
| app.seelf.subdomain   | Subdomain on which a container will be available, used as a default rule for the proxy |

Using those labels, you can easily filter resources managed by seelf, such as:

```sh
docker container ls --filter "label=app.seelf.target"
```

#### Example

Let's take an example for an application registered with the name `sandbox` and a `production` deployment on a local target with the url set to `http://docker.localhost` and a `compose.yml` file at its root:

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

- Build an image for the `app` service named `sandbox-<application id>/app:production`
- Expose the `app` service on the default subdomain `http://sandbox.docker.localhost` because that's the first service in **alphabetical order** which has **ports mappings defined**. If environment variables has been defined for the `app` service in the production environment, they will overwrite what's in the compose file
- expose the `sidecar` service on `http://sidecar.sandbox.docker.localhost` because it has port mappings too and the **production profile** is activated
- skip the `stagingonly` service because we have requested a production deployment
- run the `db` service without exposing it because it does not have port mappings defined and has such will be kept private and use any environment variables defined for the `db` service in the production environment.

## Remote targets

When configuring a remote target, you'll **have to add** the public key associated with the private one you'll be using to connect to the host to the `~/.ssh/authorized_keys` file. You can check the [Digital Ocean documentation](https://docs.digitalocean.com/products/droplets/how-to/add-ssh-keys/to-existing-droplet/#with-ssh) for more information.

## Configuration {#configuration}

When creating a target or updating its URL / provider configuration, a **configuration process** will occur to make sure the target is ready to handle deployments. This [task](/reference/jobs) will deploy the [traefik proxy](https://doc.traefik.io/traefik/) and configure it accordingly.

::: info
If you messed your server up, you can **reconfigure** a target by clicking the corresponding button on the interface. It will relaunch the configuration process.
:::

## Cleanup

Deleting a target will (if it has been configured at least once correctly) remove **everything created by seelf** on it:

- Containers
- Images
- Networks
- Volumes

::: info
If you want to delete a target and the cleanup could not be done correctly because of a particular situation, you can [cancel the cleanup job](/reference/jobs#cancellation) from the **jobs** page.
:::
