# OpenTofu/Terraform Provider ClouDNS

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.1.x (older versions may work but are entirely untested)
- [OpenTofu](https://opentofu.org/docs/intro/install/)
- [Go](https://golang.org/doc/install) >= 1.22
- [ClouDNS](https://cloudns.net) API credentials and a pre-existing DNS zone manageable by the user/sub-user associated with said credentials

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```sh
$ go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules). Please see the Go documentation for the most up to date information about using Go
modules.

To add a new dependency `github.com/author/dependency` to your OpenTofu/Terraform provider:

```sh
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Ensure that you have an API user/sub-user on ClouDNS (requires a paid subscription with reseller access).

> Note that using a sub-user which you delegate a specific zone to is a **much** safer approach and should always be your first choice

Once that is done, you must pre-create the zones you will want to manage on ClouDNS side (technically they are manageable through the API)

