# API

This part is currently being worked on. For now, only routes related to the deployment can be accessed with the API Key retrieved from the **profile page**. For more information, you can check the [`api.http` file](https://github.com/YuukanOO/seelf/blob/main/api.http) in the source code.

Every other routes use a cookie authentication.

## Allowed API access routes

The following routes are allowed with an header `Authorization: Bearer <user API Key>`.

```http
# Retrieve an app details
GET /apps/:id
# Creates a new deployment
POST /apps/:id/deployments
# Get all deployments of an app
GET /apps/:id/deployments
# Get a specific app deployment
GET /apps/:id/deployments/:number
# Redeploy a deployment
POST /apps/:id/deployments/:number/redeploy
# Promote a staging deployment to the production environment
POST /apps/:id/deployments/:number/promote
# Retrieve deployment logs
GET /apps/:id/deployments/:number/logs
```

## Triggering a deployment from your CI {#ci}

On the **New deployment** page, you can copy the `curl` command associated with the payload represented by the form. You just have to embed this command to your CI job, update it according to your needs (filling the git branch from the environment for example and retrieve the API Key from a secret) and you're good to go.

::: info
In the future, this will be simplified, see [this issue](https://github.com/YuukanOO/seelf/issues/48) for more information.
:::
