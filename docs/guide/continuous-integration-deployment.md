# Continuous Integration / Deployment

Integrating **seelf** to automatically deploy your services in your pipeline is **a breeze**.

Just use one of the following methods.

## Github Action

If your using **Github**, the easiest way to integrate **seelf** is by using the [YuukanOO/seelf-deploy-action](https://github.com/YuukanOO/seelf-deploy-action) in your workflow file.

Check its [README](https://github.com/YuukanOO/seelf-deploy-action?tab=readme-ov-file#usage-example) to know more.

## cURL

Another way to trigger a deployment is to directly use the [seelf API](/reference/api) with a program like cURL.

On the **New deployment** page, you can show the `curl` command associated with the payload represented by the form. You just have to embed this command in your CI job, update it according to your needs (filling the git branch from the environment for example and retrieve the API Key from a secret) and you're good to go.

Here is an example for the [Drone CI software](https://www.drone.io/):

```yml
kind: pipeline
type: docker
name: default
steps:
  - name: deploy
    image: curlimages/curl
    environment:
      SEELF_API_KEY:
        from_secret: seelf_api_key
    commands:
      - /bin/sh -c 'curl -X POST -H "Content-Type:application/json" -H "Authorization:Bearer $SEELF_API_KEY" -d "{\"environment\":\"staging\",\"git\":{\"branch\":\"$DRONE_BRANCH\",\"hash\":\"$DRONE_COMMIT\"}}" https://seelf.example.com/api/v1/apps/2PvP5liIhcMn59yo5q6m53QWWXM/deployments'
```

::: info
You'll need to replace the domain `seelf.example.com` and the application id `2PvP5liIhcMn59yo5q6m53QWWXM` to match your configuration.
:::
