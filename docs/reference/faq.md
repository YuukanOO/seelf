# Frequently asked question

## How does seelf know which services to expose from a `compose.yml` file? {#services-exposal}

Let's take an example for an application registered with the name `sandbox` and a `production` deployment on a local target with the url set to `http://docker.localhost`.

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

Once it has found a valid compose file, it will apply some **heuristics** to determine which services should be exposed and where.

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

- Build an image for the `app` service named `sandbox-<application id>/app:production`
- Expose the `app` service on the default subdomain `http://sandbox.docker.localhost` because that's the first service in **alphabetical order** which has **port mappings defined**. If environment variables has been defined for the `app` service in the production environment, they will overwrite what's in the compose file
- expose the `sidecar` service on `http://sidecar.sandbox.docker.localhost` because it has port mappings too and the production profile is activated
- skip the `stagingonly` service because we have requested a production deployment
- run the `db` service without exposing it because it does not have port mappings defined and has such will be kept private and use any environment variables defined for the `db` service in the production environment.

To expose those services, seelf will add the needed traefik labels based on the target's url you've set when configuring it. If your domain starts with `https://`, Let's encrypt certificates will be generated for you.

## Docker labels appended by seelf

To identify which resources are managed by seelf, some **docker labels** are appended during the deployment process.

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

## Integrating seelf in your CI

See the [API related page](/reference/api#ci) for more information on how to do it.
