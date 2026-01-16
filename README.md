# go-mod-size

A tiny CLI tool for analyzing the source code size of all dependencies in a Go project.

The size of of dependencies source code affects Go builds that don't have these dependencies pre-cached and you generally want to keep it small even if you have cache enabled and preserved everywhere, including CI pipelines, CI Docker builds, local Go builds, local Docker builds, etc.

To see the effect dependencies have on the Go binary size, use [go-size-analyzer](https://github.com/Zxilly/go-size-analyzer) instead.

## 📦 Installation

```bash
go install github.com/orsinium-labs/go-mod-size@latest
```

## 🔧 Usage

```bash
go-mod-size .
```
