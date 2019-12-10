# Go zabbix api

[![GoDoc](https://godoc.org/github.com/claranet/go-zabbix-api?status.svg)](https://godoc.org/github.com/claranet/go-zabbix-api) [![Build Status](https://travis-ci.org/claranet/go-zabbix-api.svg?branch=master)](https://travis-ci.org/AlekSi/zabbix??branch=master)

This Go package provides access to Zabbix API.

Tested on Zabbix 3.2, 3.4, 4.0, 4.2 and 4.4, but should work since 2.0 version.

This package aims to support multiple zabbix resources from its API like trigger, application, host group, host, item, template..

## Install

Install it: `go get github.com/claranet/go-zabbix-api`

## Getting started

```
package main

import (
	"fmt"

	"github.com/claranet/go-zabbix-api"
)

func main() {
	user := "MyZabbixUsername"
	pass := "MyZabbixPassword"
	api := zabbix.NewAPI("http://localhost/api_jsonrpc.php")
	api.Login(user, pass)

	res, err := api.Version()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Connected to zabbix api v%s\n", res)
}

```

## Run test

You should run tests before using this package – Zabbix API doesn't match documentation in few details, which are changing in patch releases. Tests are not expected to be destructive, but you are advised to run them against not-production instance or at least make a backup.

    export TEST_ZABBIX_URL=http://localhost:8080/zabbix/api_jsonrpc.php
    export TEST_ZABBIX_USER=Admin
    export TEST_ZABBIX_PASSWORD=zabbix
    export TEST_ZABBIX_VERBOSE=1
    go test -v

`TEST_ZABBIX_URL` may contain HTTP basic auth username and password: `http://username:password@host/api_jsonrpc.php`. Also, in some setups URL should be like `http://host/zabbix/api_jsonrpc.php`.

## References

Documentation is available on [godoc.org](https://godoc.org/github.com/claranet/go-zabbix-api).
Also, Rafael Fernandes dos Santos wrote a [great article](http://www.sourcecode.net.br/2014/02/zabbix-api-with-golang.html) about using and extending this package.

License: Simplified BSD License (see LICENSE).
