# Updating

This procedure depends on the method you choose when [installing seelf](/guide/installation) initially.

::: warning
When switching from a major version to another one (ex. `v1.x.x` to `v2.x.x`), check the [Releases page](https://github.com/YuukanOO/seelf/releases) for additional instructions.
:::

## With Compose

Go where the initial `compose.yml` file has been created and run:

```sh
docker compose pull && docker compose up -d
```

::: info
If you use a specific image version, you must update the tag in the compose file before running the above command.
:::

## With Docker

Since the configuration has been saved in the volume `seelfdata` the first time you launched seelf, you can omit the settings here unless they have changed, in this case, you should probably check the configuration file in `seelfdata/conf.yml`.

```sh
docker pull yuukanoo/seelf && docker rm $(docker stop $(docker ps -a -q --filter="ancestor=yuukanoo/seelf")) && docker run -d -v "/var/run/docker.sock:/var/run/docker.sock" -v "seelfdata:/seelf/data" -v "seelfssh:/root/.ssh" -p "8080:8080" yuukanoo/seelf
```

## From sources

Simply build the application again with the latest sources and you're good to go.
