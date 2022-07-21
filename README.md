curly
=====

[curl-to-go](https://github.com/mholt/curl-to-go) port with the power of Golang.

This is fun project which converts `curl` command to Golang code with ability of its further execution.

The idea is to re-use existing tools as much as possible in Golang-ecosystem.

### Dependencies

It would be impossible to build this utility without next pieces of software:

| Library          | What it does?                                                       |
|------------------|---------------------------------------------------------------------|
| [mholt/curl-to-go](https://github.com/mholt/curl-to-go) | The original Javascript converter code                              |
 |[mholt/json-to-go](https://github.com/mholt/json-to-go) | The auxiliary Javascript library for go structs                     |
|[jerrybendy/url-search-params-polyfill](https://github.com/jerrybendy/url-search-params-polyfill) | The missing Javascript bit from V8 engine ECMAScript implementation |
|[dop251/goja](https://github.com/dop251/goja) | ECMAScript/JavaScript engine in pure Go                             |
|[traefik/yaegi](https://github.com/traefik/yaegi) | Elegant Go Interpreter written in Go                                |
|[urfave/cli](https://github.com/urfave/cli) | CLI-app framework for Go                                            |


### How does it work?

`curly` itself provides just "glue"-code for all the parts together in form of a CLI-application.

Processing might be explained by following diagram:

![diagram](./curly.drawio.png)

The tool reads `curl` command from `STDIN` and executes converter `curlToGo` on V8 engine **Goja**.

Received go code enhanced with missing imports and beautified.

Depending on flags code could be dumped or interpreted in runtime via **Yaegi**.


### Installation

Currently, distribution is available in form of a binary executable on [Releases]() page for Linux and Mac OS.

Binary could be added to your `bin` directory.


### Usage
1. Redirect `curl` command from `STDOUT` and run it with default params:

    ```shell  
    echo "curl -X GET https://example.com" | curly
    ```
2. Redirect `curl` command from `STDOUT` and dump generated go code without execution:
    ```shell
    echo "curl -X GET https://example.com" | curly -d
    ```
3. Read `curl` command from clipboard and run generated code in 50 requests with 5 concurrency:
    ```shell
    xclip -o | curly -r 50 -c 5
    ```
4. Read `curl` command from file and run generated code in 10 requests with 1 concurrency and
   sleep duration of 1 second:
   ```shell
   cat curl.txt | curly -r 10 -s 1s
   ```


### Build

It's possible to build binary by your own:

```shell
make build
```