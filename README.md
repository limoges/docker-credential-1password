# docker-credential-1password
> Use docker credentials stored in 1password

Docker Credential 1password is a [docker credential helper] which pulls
credentials from 1password.

## Usage

### Prerequisites

- You must have installed `op`, the [1Password CLI].
- You must have installed `docker-credential-1password`.

## Configuration

### Adding a credential to 1password

The credential MUST define the fields:
- `URL`
- `username`
- `password`

These fields correspond to these `docker login` commands:
```bash
# using password flag
docker login --password ${password} --username ${username} ${URL}
# using password-stdin
echo ${password} | docker login --username ${username} --password-stdin ${URL}
```

`docker-credential-1password` will search for credentials using these filters
```bashckkk
op item list \
    --vault ${DOCKER_CREDENTIAL_1PASSWORD_VAULT:-Docker} \
    --categories ${DOCKER_CREDENTIAL_1PASSWORD_CATEGORY:-Server} \
    --tags ${DOCKER_CREDENTIAL_1PASSWORD_TAG:-Docker Credentials}
```

For example, to create a docker credential from your `GITHUB_TOKEN`:
```bash
op item create \
    --vault Docker \
    --category Server \
    --tags docker-credential-1password \
    --title ghcr.io \
    "username=USERNAME" \
    "password=${GITHUB_TOKEN}" \
    "URL=ghcr.io"
```

### Configuring docker

You can use registry-specific credential helpers.

For example, this configuration will search 1password for stored credentials:
```bash
cat $HOME/docker/config.json
{
	"credHelpers": {
        "ghcr.io": "1password",
        "quay.io": "1password"
	},
	"auths": {
		"https://index.docker.io/v1/": {}
	},
	"credsStore": "osxkeychain"
}
```

[docker credential helper]:https://docs.docker.com/reference/cli/docker/login/#credential-helpers
[1Password CLI]:https://developer.1password.com/docs/cli/get-started/
