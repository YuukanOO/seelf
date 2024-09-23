# Targets

Targets represents an **host** where your deployments will be exposed. When configuring an application, you choose on which target a specific environment should be deployed.

::: info
For now, only one target per host is allowed.
:::

## Proxy configuration {#proxy}

When declaring a target, you must choose how the proxy (needed to make your services available from the outside world) should be managed:

- **Automatic**: **seelf** will deploy and configure a [traefik](https://traefik.io/traefik/) proxy on the target. Services urls will be automatically generated based on the [target's url](#url) and [service file](/reference/providers/docker#exposing-services) when deploying. Exposed services will also join the proxy network.
- **Manual**: you're in charge of **everything** related to services exposure. **seelf** will deploy services on this target without attempting to expose them in any way.

### Url

If the target manages the proxy itself, this url **determines where your applications will be made available**. It should be a **root url** as applications will use subdomains on it.

The scheme associated with this url (`http` or `https`) will determine if certificates should be generated or not.

## Providers {#providers}

You must choose one provider kind when creating a target. Some providers have specific parameters for you to configure how things work. See the [providers reference](/reference/providers) for more informations.

::: warning
Whatever provider you choose, you should make sure your **DNS is correctly configured with a wildcard redirecting to the target host**, the [DigitalOcean procedure](https://docs.digitalocean.com/glossary/wildcard-record/) can be applied to your specific provider.
:::

## Remote targets

When configuring a remote target, you'll **have to add** the public key associated with the private one you'll be using to connect to the host to the `~/.ssh/authorized_keys` file. You can check the [Digital Ocean documentation](https://docs.digitalocean.com/products/droplets/how-to/add-ssh-keys/to-existing-droplet/#with-ssh) for more information.

## Configuration {#configuration}

When creating a target, updating its url / provider configuration or when new custom entrypoints should be created to handle custom ports, a **configuration process** will occur to make sure the target is ready to handle deployments. This [task](/reference/jobs) will deploy the needed infrastructure on the target.

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
