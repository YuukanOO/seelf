# Targets

Targets represents an **host** where your deployments will be exposed. When configuring an application, you choose on which target a specific environment should be deployed.

::: info
For now, only one target per host is allowed.
:::

## Providers {#providers}

You must choose one provider kind when creating a target. Some providers have specific parameters for you to configure how things work.

### Docker

The only supported provider for now. Uses Docker Compose to launch your services by looking in the project root for specific files, see [services exposal](/reference/faq#services-exposal).

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
