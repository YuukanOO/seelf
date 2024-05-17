# Deployments

Created from an [application](/reference/applications) for a specific [environment](/reference/applications#environments), represents an actual deployment on a [target](/reference/targets).

## Sources {#sources}

Deployments can be created from a number of sources.

### Archive (`tar.gz`)

An archive containing your project files to be deployed.

### Raw file

A raw file. For example, a `compose.yml` content when using the [Docker provider](/reference/providers/docker).

### Git

A valid **branch** and an optional specific **commit** if the application has been configured with a version control system.
