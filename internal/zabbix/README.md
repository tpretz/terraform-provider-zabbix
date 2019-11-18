go zabbix api[![Build Status](https://travis-ci.org/claranet/go-zabbix-api.svg?branch=master)](https://travis-ci.org/AlekSi/zabbix??branch=master)
======

This Go package provides access to Zabbix API.

Tested on Zabbix 3.2 but should works since 2.0 version.

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
