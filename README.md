[![Build Status](https://travis-ci.org/zc310/log.svg)](https://travis-ci.org/zc310/log)
[![GoDoc](https://godoc.org/github.com/zc310/log?status.svg)](http://godoc.org/github.com/zc310/log)
[![Go Report](https://goreportcard.com/badge/github.com/zc310/log)](https://goreportcard.com/report/github.com/zc310/log)

# log
golang log


#### Example



```go
package main

import (
	"github.com/zc310/log"
)

func main() {
	log.SetPath("/tmp/")
	log.Info("a", 1, 2, []string{"a", "b", "c"})
}
```