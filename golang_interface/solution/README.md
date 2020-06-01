# golang_interface

[中文版本](README_zh.md)

This challenge is inspired by [gomium](https://github.com/google/google-ctf/tree/master/2019/finals/pwn-gomium) from google ctf 2019 finals.

## Description

https://golang-interface.rctf2020.rois.io

```go
file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.AllErrors)
if err != nil {
    return nil, errors.New("Syntax error")
}
if len(file.Imports) > 0 {
    return nil, errors.New("Imports are not allowed")
}

// go build -buildmode=pie and run for 1s...
```

## Solution

The challenge receives a Go source code without any imports. You can see Stalkr's [Golang data races to break memory safety](https://blog.stalkr.net/2015/04/golang-data-races-to-break-memory-safety.html) for more detailed analysis.

See the full [exploit](exploit.go) here.

## Thank you for your participation, see you next year @ RCTF2021!

