# Configuration

**seelf** can be configured from a **yaml** file and/or **environment variables**. Environment variables will always take precedence over the configuration file.

When configured using a file, you can provide it with the `-c <your_file.yml>` or `--config <your_file.yml>`.

If the configuration file does not exist, **it will be created** with the initial configuration.

::: info
Environment variables can also be defined in a `.env` or `.env.local` file in the working directory when launching seelf.
:::

## Reference

| yaml path / env name(s)                                 | Description                                                                                                                                                                                                                                                 | Default value                         |
| ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| log.level<br>LOG_LEVEL                                  | Log level to use (info, warn or error)                                                                                                                                                                                                                      | info                                  |
| log.format<br>LOG_FORMAT                                | Format of the logs (json, console)                                                                                                                                                                                                                          | console                               |
| data.path<br>DATA_PATH                                  | Where data produced by seelf will be saved (deployment artifacts, logs, local db, …)                                                                                                                                                                        | ~/.config/seelf                       |
| data.deployment_dir_template<br>DEPLOYMENT_DIR_TEMPLATE | [Go template](https://pkg.go.dev/text/template) determining the directory where the build will occur (use <code v-pre>{{ .Number }}-{{ .Environment }}</code> if you want to keep all application deployment sources for example)                           | <code v-pre>{{ .Environment }}</code> |
| source.archive.max_size<br>SOURCE_ARCHIVE_MAX_SIZE      | Maximum file size allowed, suffixes allowed: b, kb, mb and gb (such as `20mb`)                                                                                                                                                                              | 32mb                                  |
| http.host<br>HTTP_HOST                                  | Host to listen to                                                                                                                                                                                                                                           | 0.0.0.0                               |
| http.port<br>HTTP_PORT,PORT                             | Port to listen to                                                                                                                                                                                                                                           | 8080                                  |
| http.secure<br>HTTP_SECURE                              | Wether or not the web server is served over https. If omitted, determine this information from the `EXPOSED_ON` variable. It controls wether or not cookie are set with the `Secure` flag and the scheme used on the `Location` header of created resources | false                                 |
| http.secret<br>HTTP_SECRET                              | Secret key to use when signing cookies                                                                                                                                                                                                                      | &lt;generated if empty&gt;            |
| runners<br>RUNNERS_CONFIGURATION                        | Array of runners configuration, [see below](#runners) for more information                                                                                                                                                                                  |                                       |
| -<br>ADMIN_EMAIL                                        | Email of the first user account to create (mandatory if no user account exists yet)                                                                                                                                                                         |                                       |
| -<br>ADMIN_PASSWORD                                     | Password of the first user account to create (mandatory if no user account exists yet)                                                                                                                                                                      |                                       |
| -<br>EXPOSED_ON                                         | Url at which the seelf container [will be exposed](/guide/installation#exposing-seelf) and default target url. In the form `<url scheme>://<container name>@<default target url>`                                                                           |                                       |

### Runners configuration {#runners}

::: warning
Since version `2.5.0`, the runners configuration has changed. **seelf** will try its best to understand your configuration, but you **should** use the newer format:

```yml
runners:
  poll_interval: 4s // [!code --]
  deployment: 4 // [!code --]
  cleanup: 2 // [!code --]
  - poll_interval: 4s // [!code ++]
    count: 4 // [!code ++]
    jobs: // [!code ++]
      - deployment.command.deploy // [!code ++]
  - poll_interval: 4s // [!code ++]
    count: 2 // [!code ++]
```

:::

You have full control over how [background jobs](/reference/jobs) are processed as long as each application job is at least handled by one runner. Each **runner** will run in parallel and poll for jobs assigned to it, passing them to their respective **workers**.

The goal of this configuration is to tell **seelf** how many runners should live to treat each job and at which rate they poll for new ones. With this level of configuration, you can make seelf suit your particular infrastructure.

| yaml path     | Description                                                                                                                                                                                                                                                                                    |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| poll_interval | Interval at which [background jobs](/reference/jobs) are picked. Should be parsable by [time.ParseDuration](https://pkg.go.dev/time#ParseDuration)                                                                                                                                             |
| count         | Number of workers, which means how many parallel jobs are allowed inside this runner                                                                                                                                                                                                           |
| jobs          | Array of job names handled by this runner. The last runner definition could leave it empty to handle all jobs not already handled), should be one of `deployment.command.deploy`, `deployment.command.cleanup_app`, `deployment.command.configure_target`, `deployment.command.cleanup_target` |

::: info Configuring runners using the environment variable
You can also use the `RUNNERS_CONFIGURATION` variable to configure runners using the format `<poll_interval>;<count>;<job names comma separated>` with runners separated by `|`.

Here is an example representing the default configuration:

```sh
RUNNERS_CONFIGURATION="4s;4;deployment.command.deploy|4s;2;" seelf serve
```

:::

If not specified, the default configuration is as follow:

```yml
runners:
  - poll_interval: 4s
    count: 4
    jobs:
      - deployment.command.deploy
  - poll_interval: 4s
    count: 2
```

Meaning **deployment jobs** will be processed by **4** workers (hence deploying a maximum of **4** apps at a time), checking every **4 seconds**, and every other jobs will be handled by **2** workers.

With this in place, deployment jobs will not prevent **non-related jobs** to be processed.
