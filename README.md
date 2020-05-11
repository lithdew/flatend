# flatend

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](LICENSE)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lithdew/flatend)
[![Discord Chat](https://img.shields.io/discord/697002823123992617)](https://discord.gg/HZEbkeQ)

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