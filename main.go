package main

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/limoges/docker-credential-1password/internal/onepassword"
)

func main() {
	credentials.Serve(onepassword.NewHelper())
}
