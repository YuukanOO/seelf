# Docker provider

Uses [Docker Compose](https://docs.docker.com/compose/) to launch your services by looking in the project root for specific files and configure an appropriate [traefik proxy](https://doc.traefik.io/traefik/) to expose your services.

::: warning
[Docker >= (v18.0.9) must be installed](https://docs.docker.com/get-docker/) on the target!

Docker engine `v1.41` can sometimes cause issues when attaching multiple networks on container creation. If you have any issue, consider updating.
:::

## Files looked at

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

## Exposing services

Once a valid compose file has been found and **only if** the target [manages the proxy itself](/reference/targets#proxy), **seelf** will apply some **heuristics** to determine which services should be exposed and where.

It will consider any service with **port mappings** to be exposed.

::: info Why relying on **ports mappings**?
When working on a local compose stack, you make **services available by defining ports mappings**. By using this **heuristic**, we make the transition from local to remote a breeze.
:::

::: warning Exposing TCP and UDP ports
When you don't define a specific protocol, **compose** falls back to `tcp` when reading the project file. When exposing services, the distinction between `http` and `tcp` is mandatory for the proxy to work as intended and expose your services appropriately.

To make the distinction, **seelf** rely on how ports are defined in the compose file:

- **Without a specific protocol** (ex `- "8080:80"`), seelf will assumes it should use an `http` router.
- **With a specific protocol** (ex `- "8080:80/tcp"` or `- "8080:80/udp"`), it will use the router associated: `tcp` or `udp`.

When custom entrypoints are found (multiple `http` ports on the same container or `tcp` / `udp` ports), **seelf does not rely on the actual published ports defined** since it can't assume they will be available on the [target](/reference/targets).

Instead, it will spawn a tiny container to **find available ports of the specified protocols** and map them to your entrypoints. This process **only happens the first time your custom entrypoints are seen** and as long as you do not change the **service name**, **container port** and **protocol**.

Using this tiny rule, **seelf** can determine the router to use correctly and your compose file **still works locally**.
:::

The first service in **alphabetical order** using an `http` router will take the [default application subdomain](/reference/applications#environments). Every other services exposed will be on a subdomain of that default one.

If some services uses custom entrypoints, the target will be [reconfigured](/reference/targets#configuration) automatically to **make them available**.

::: warning Proxy unavailability
Currently, the proxy to handle the default http entrypoints and custom ones is **shared** meaning there is a **tiny unavailability** when new entrypoints should be exposed (the first time they are seen or when they are no more needed).

In the future, it will be possible to deploy a sidecar proxy specifically for custom entrypoints to prevent this, see [this issue](https://github.com/YuukanOO/seelf/issues/62).
:::

## Labels appended by seelf

To identify which resources are managed by seelf, some **docker labels** are appended during the deployment process. Some labels such as `app.seelf.application`, `app.seelf.target`, `app.seelf.environment` and `app.seelf.custom_entrypoints` are appended to each resources: container, networks, volumes and images built while the others labels are only appended to the container.

| Name                         | Description                                                                            |
| ---------------------------- | -------------------------------------------------------------------------------------- |
| app.seelf.exposed            | Only used to identify the seelf container when exposing it through a local target      |
| app.seelf.application        | ID of the application                                                                  |
| app.seelf.environment        | [Environment](/reference/applications#environments) of the resource                    |
| app.seelf.target             | ID of the target on which the container must be exposed                                |
| app.seelf.subdomain          | Subdomain on which a container will be available, used as a default rule for the proxy |
| app.seelf.custom_entrypoints | Appended on a service which uses custom entrypoints                                    |

Using those labels, you can easily filter resources managed by seelf, such as:

```sh
docker container ls --filter "label=app.seelf.target"
```

## Example

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
    # if you wish to expose the db service, don't forget the /tcp !
    # ports:
    #   - "5432:5432/tcp"
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

::: info
If you uncomment the `db` ports part, the db will be exposed using a custom entrypoint and a port will be allocated to handle it.
:::
