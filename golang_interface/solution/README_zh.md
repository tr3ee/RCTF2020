# golang_interface

[English version](README.md)

本题是受到google ctf 2019 finals中[gomium](https://github.com/google/google-ctf/tree/master/2019/finals/pwn-gomium)的启发而写的

## 题目描述

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

## 解题思路

题目接收一个Go源码文件，但是不允许任何外部库的导入. 你可以直接看原作者Stalkr的详细分析[Golang data races to break memory safety](https://blog.stalkr.net/2015/04/golang-data-races-to-break-memory-safety.html)

点击查看完整的[利用代码](exploit.go)

## 感谢你的参与，期待明年RCTF2021再见！