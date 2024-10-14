# Migration

Most of the time, you don't have to act manually when upgrading **seelf**. However, sometimes, **breaking changes** must be introduced. This page lists major migrations.

## From v1 to v2 {#v2}

::: warning
This migration assumes you have **at least one application you care about** on your seelf instance. If you do not have any application yet, you **should** remove everything and go back to [installing seelf](/guide/installation) since it will be easier.
:::

The **seelf** `v2` introduces some breaking changes. When coming from the `v1.x.x`, you will need to take manual actions.

**Every resources** deployed by **seelf** `v1.x.x` will not be compatible and will **need to be redeployed**, including the default proxy.

Here is a breakdown of actions needed:

1. **Stop** all existing containers
1. **Update the compose** file (only if installed via Compose)
1. **Create** the new gateway network
1. **Update and launch** seelf
1. **Update** the newly created (by the migration script) local target url to match the old `BALANCER_DOMAIN`
1. **Migrate old balancer data** (certificates) to the new one (skip if no certificates have been generated)
1. Update the seelf compose again (optional, see below)
1. **Redeploy** every apps
1. **Migrate application data** (if you wish to keep them)
1. **Prune** unneeded resources

### Stop existing containers

Before doing anything, start by stopping all containers:

```sh
docker stop $(docker ps -q)
```

### Update the seelf compose file

If you chose the recommended method to install **seelf**, you should have a `compose.yml` file.

#### Without traefik labels

If you do not expose seelf using the internal proxy, the update is straightforward.

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf
    environment:
      - BALANCER_DOMAIN=https://your.domain // [!code --]
      - ACME_EMAIL=youremail@provider.com // [!code --]
      - SEELF_ADMIN_EMAIL=admin@example.com // [!code --]
      - SEELF_ADMIN_PASSWORD=admin // [!code --]
      - ADMIN_EMAIL=admin@example.com // [!code ++]
      - ADMIN_PASSWORD=admin // [!code ++]
      # (...) other variables left unchanged
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - seelfssh:/root/.ssh // [!code ++]
      - seelfdata:/seelf/data

volumes:
  seelfdata:
  seelfssh: // [!code ++]
```

#### With traefik labels

The procedure used to expose seelf [has been simplified](/guide/installation#exposing-seelf) but the migration is a bit tricky. A default target will be created when launching seelf `v2.x.x` for the first time if it doesn't exist and this target will have an URL of `http://docker.localhost`.

For this to work, we keep the `traefik.http.routers.seelf.rule` label for the duration of the migration and we remove the `https` stuff such as `HTTP_SECURE` and the cert resolver to make sure we can connect to seelf without `https`. During the migration, you will access seelf using `http` only.

::: warning
Before updating the default local target URL, you'll have to trust invalid certificates and access the seelf instance without `https`.
:::

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf
    container_name: seelf # Should match the user portion of EXPOSED_ON when exposing seelf using a local target // [!code ++]
    environment:
      - BALANCER_DOMAIN=https://your.domain // [!code --]
      - ACME_EMAIL=youremail@provider.com // [!code --]
      - SEELF_ADMIN_EMAIL=admin@example.com // [!code --]
      - SEELF_ADMIN_PASSWORD=admin // [!code --]
      - ADMIN_EMAIL=admin@example.com // [!code ++]
      - ADMIN_PASSWORD=admin // [!code ++]
      - EXPOSED_ON=https://seelf@your.domain // [!code ++]
      - HTTP_SECURE=false # false is important during the migration // [!code ++]
      # (...) other variables left unchanged
    labels:
      - app.seelf.exposed=true // [!code ++]
      - app.seelf.subdomain=seelf // [!code ++]
      - traefik.enable=true // [!code --]
      - traefik.http.routers.seelf.rule=Host(`seelf.your.domain`) # This line can be removed AFTER the migration
      - traefik.http.routers.seelf.tls.certresolver=seelfresolver // [!code --] # Not needed anymore and we need to access seelf without certificates during the migration process
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - seelfssh:/root/.ssh // [!code ++]
      - seelfdata:/seelf/data

volumes:
  seelfdata:
  seelfssh: // [!code ++]

networks: // [!code --]
  default: // [!code --]
    name: seelf-public // [!code --]
```

### Create the new gateway network

When migrating to the `v2` version, a default target will be created to preserve your applications. This target have a fixed id. **If you expose seelf using the local target**, you **have to** create the local network before launching seelf so that the container can join the local target network at startup.

```sh
docker network create seelf-gateway-2brudqnyrelmqyh9gflqv1s0cqv --label com.docker.compose.network=default --label com.docker.compose.project=seelf-internal-2brudqnyrelmqyh9gflqv1s0cqv --label app.seelf.target=2bRUdQnyRELMqyh9gFLQV1s0cqv
```

### Update and launch seelf

Now that our compose file is ready, pull the latest image and relaunch seelf:

::: info
If you can't connect to your seelf instance after that, try to clear the cookie storage, it may be needed if the `HTTP_SECURE` variable have changed.
:::

```sh
docker compose pull && docker compose up -d
```

### Update the local target url

Login to your seelf instance and go to the new targets page. You should see a local target. **Update the URL field** to match your old `BALANCER_DOMAIN` variable and click **Save**.

**Wait** for the configuration to succeed, it will create a volume to hold certificates if it starts with `https://`.

### Migrate old certificates {#certificates}

If you use `https://`, you must transfer the old certificates to the new target volume to avoid soliciting Let's encrypt again (and it will fail if your certificates are recent).

To do so, we will use the `docker cp` command, to transfer data from our old proxy container to the new one. Since we cannot transfer directly from one container to another, we will transfer first to our host.

```sh
docker cp seelf-internal-balancer-1:/letsencrypt ./letsencrypt && docker cp ./letsencrypt/. seelf-internal-2brudqnyrelmqyh9gflqv1s0cqv-proxy-1:/letsencrypt/ && rm -rf ./letsencrypt/ && docker restart seelf-internal-2brudqnyrelmqyh9gflqv1s0cqv-proxy-1
```

### Update the seelf compose (only if it was exposed using the default proxy and using https)

```yml
services:
  web:
    restart: unless-stopped
    image: yuukanoo/seelf
    container_name: seelf
    environment:
      - ADMIN_EMAIL=admin@example.com
      - ADMIN_PASSWORD=admin
      - EXPOSED_ON=https://seelf@your.domain
      - HTTP_SECURE=false // [!code --]
      - HTTP_SECURE= # Go back to its default behavior based on EXPOSED_ON scheme // [!code ++]
      # (...) other variables left unchanged
    labels:
      - app.seelf.exposed=true
      - app.seelf.subdomain=seelf # Here your can update the subdomain used by seelf if you wish
      - traefik.http.routers.seelf.rule=Host(`seelf.your.domain`) // [!code --]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - seelfssh:/root/.ssh
      - seelfdata:/seelf/data

volumes:
  seelfdata:
  seelfssh:
```

### Redeploy every apps

Use the web UI to redeploy every apps you want. This is needed because project name and labels have been changed to make sure no conflict can occurs in the future.

### Migrate application data

Since project names have changed, volumes have too. You can use the same command as for [migrating old certificates](#certificates) to transfer application data from the old container to the new one.

::: info
Since the user base of the `v1` is still relatively small, I did not take the time to write a migration script to automate this part. If you did write something, feel free to contribute to this documentation.
:::

```sh
# Find the source and destination container id, let's refer to them as SRC_ID and DEST_ID
docker ps -a
# List every path we need to transfer
docker inspect --format='{{ range .HostConfig.Mounts }}{{ .Target }}{{ end }}'  SRC_ID
# Retrieve old data to the host
docker cp SRC_ID:/path/from/above ./tmp
docker cp ./tmp/. DEST_ID:/path/from/above/
# Restart the container to make sure
docker restart DEST_ID
# Remove temp directory from the host
rm -rf ./tmp
```

### Prune unneeded resources

Now that everything is good, we can safely prune our old resources:

```sh
docker system prune -af --volumes
docker volume prune -af # If docker engine < v1.42, remove the `a` flag
```

Everything should be good now, and if it's not, feel free to [open an issue](https://github.com/YuukanOO/seelf/issues).
