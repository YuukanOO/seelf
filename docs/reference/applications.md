# Applications

Defines a new **stack of services** exposed on a [target subdomain](/reference/targets). The name will be used as a subdomain and should be unique **per target and environment**.

## Multiple exposed services

When you define a complete stack for your application, you may have multiple services exposed. This is totally allowed by **seelf**. When [identifying services](/reference/providers/docker#exposing-services) which must be exposed, the first one in **alphabetical order** will become the **default service** and take the default subdomain.

Other services will be exposed using a subdomain on the default one.

## Environments {#environments}

Only 2 environments are managed by **seelf** at the moment: **production** and **staging**.

::: info
Any updates on an application environment will trigger a redeploy of the latest deployment.
:::

Each one of those environment can be deployed on a separate target and configured appropriately with specific environment variables.

::: warning
When you update an environment target, if at least one successful deployment have been made on the old target, a cleanup job will be queued, meaning all resources for this specific application and environment will be **deleted** on the old target.

This prevent a target from having dangling applications.
:::

### Production

Represents the main environment. The **default service** will be exposed on `<target scheme>://<app name>.<target root url>`. Any additional exposed services will add another level such as `<target scheme>://<service name>.<app name>.<target root url>`.

### Staging

For the staging environment, a `-staging` suffix is added to the application name: `<target scheme>://<app name>-staging.<target root url>`.

## Cleanup

Deleting an application will (if at least one deployment has been successful on a target) remove **everything created by seelf** on it:

- Containers
- Images
- Networks
- Volumes

::: info
If you want to delete an application and the cleanup could not be done correctly because of a particular situation, you can [cancel the cleanup job](/reference/jobs#cancellation) from the **jobs** page.
:::
