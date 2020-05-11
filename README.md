# flatend

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](LICENSE)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lithdew/flatend)
[![Discord Chat](https://img.shields.io/discord/697002823123992617)](https://discord.gg/HZEbkeQ)

## Features

1. Supports decoding/encoding requests and responses to/from JSON, YAML, TOML, CSV, or Gob.
2. Supports live-editing, staging, and deployment of APIs.
3. Supports load-balancing and TLS out-of-the-box.

## Request Lifecycle

```asciidoc
+-------+                       
|Request|                       
+-------+       -               
|                               
v                               
+------------------------------+
|Negotiate codec for           |
|decoding input/encoding output|
+------------------------------+
|                               
v                               
+--------------------+          
|Grab                |          
|1) route parameters |          
|2) query parameters |          
|3) header parameters|          
+--------------------+          
|                               
v                               
+----------------------+        
|Decode body parameters|        
|using codec           |        
+----------------------+        
|                               
v                               
+-----------------------------+ 
|Execute handler w/ parameters| 
+-----------------------------+ 
|                               
v                               
+---------------------+         
|Encode handler output|         
|using codec          |         
+---------------------+         
|                               
v                               
+--------+                      
|Response|                      
+--------+                      
```

## Benchmarks

```
$ cat /proc/cpuinfo | grep 'model name' | uniq
model name : Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz

$ wrk -d 10s -t 8 http://localhost:44444
Running 10s test @ http://localhost:44444
  8 threads and 10 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   107.92us  272.76us  15.57ms   97.49%
    Req/Sec    11.69k   710.54    17.09k    86.19%
  935023 requests in 10.10s, 421.78MB read
  Non-2xx or 3xx responses: 935023
Requests/sec:  92609.13
Transfer/sec:     41.77MB
```