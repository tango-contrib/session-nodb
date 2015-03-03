session-nodb [![Build Status](https://drone.io/github.com/tango-contrib/session-nodb/status.png)](https://drone.io/github.com/tango-contrib/session-nodb/latest) [![](http://gocover.io/_badge/github.com/tango-contrib/session-nodb)](http://gocover.io/github.com/tango-contrib/session-nodb)
======

Session-nodb is a store of [session](https://github.com/tango-contrib/session) middleware for [Tango](https://github.com/lunny/tango) stored session data via [nodb](http://github.com/lunny/nodb). 

## Installation

    go get github.com/tango-contrib/session-nodb

## Simple Example

```Go
package main

import (
    "github.com/lunny/tango"
    "github.com/tango-contrib/session"
    "github.com/tango-contrib/session-nodb"
)

type SessionAction struct {
    session.Session
}

func (a *SessionAction) Get() string {
    a.Session.Set("test", "1")
    return a.Session.Get("test").(string)
}

func main() {
    o := tango.Classic()
    o.Use(session.New(session.Options{
        Store: redistore.New(nodbstore.Options{
                Path:    "./nodbstore",
                DbIndex: 0,
                MaxAge:  30 * time.Minute,
            }),
        }))
    o.Get("/", new(SessionAction))
}
```

## Getting Help

- [API Reference](https://gowalker.org/github.com/tango-contrib/session-nodb)
