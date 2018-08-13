# zlog
zlog - A reliable, high-performance, flexsible, clear-model golang logging library.

## Features
* json conf file
* support rotate by year/month/day/hour
* detached file for warning/fatal level
* record file name and line number

## log.json
```
{
    "LogLevel" : "info",

    "FileWriter" : {
        "On": true,

        "LogPath" : "./log/service.log.info",
        "RotateLogPath" : "./log/service.log.info.%Y%M%D%H",

        "WfLogPath" : "./log/service.log.wf",
        "RotateWfLogPath" : "./log/service.log.wf.%Y%M%D%H",

        "PublicLogPath" : "./log/public.log",
        "RotatePublicLogPath" : "./log/public.log.%Y%M%D%H"
    },

    "ConsoleWriter" : {
        "On" : false
    }
}
```

## go test -bench=. log_test.go -benchmem
```
BenchmarkXlog4go-4        	  500000	      3293 ns/op	     380 B/op	      14 allocs/op
BenchmarkZlog-4           	 1000000	      2148 ns/op	     145 B/op	       6 allocs/op
BenchmarkTestZlogHigh-4   	 1000000	      1547 ns/op	     325 B/op	       1 allocs/op
```

```
package main

import (
	"testing"

	"go.intra.xiaojukeji.com/engine/zlog"

	logger "go.intra.xiaojukeji.com/engine/doom-common-go/xlog4go"
)

func init() {

	if err := logger.SetupLogWithConf("./log.json"); err != nil {
		panic(err)
	}
	if err := zlog.SetupLogWithConf("./log.json"); err != nil {
		panic(err)
	}

}
func BenchmarkXlog4go(b *testing.B) {
	for i := 0; i < b.N; i++ {
		foo1()
	}
}
func BenchmarkZlog(b *testing.B) {
	for i := 0; i < b.N; i++ {
		foo2()
	}
}
func BenchmarkTestZlogHigh(b *testing.B) {
	for i := 0; i < b.N; i++ {
		foo3()
	}
}
func foo1() {
	logger.Info("abc age=%d name=%s addr=%s company=%s angle=%f", 100, "hello", "beijing", "didi", 3.1415926)
}

func foo2() {
	zlog.Info("abc age=%d name=%s addr=%s company=%s angle=%f", 100, "hello", "beijing", "didi", 3.1415926)
}
func foo3() {
	zlog.HighInfo("abc", zlog.Int("age", 100), zlog.String("name", "liu"), zlog.String("addr", "beijing"), zlog.String("company", "didi"), zlog.Float64("angle", 3.1415926))
}
```
