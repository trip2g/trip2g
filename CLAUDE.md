Read instructions of common patterns in docs/instructions.md

## Golang

Don’t write

```golang
if err := ...; err != nil
```

Always use two lines:

```golang
err = ...
if err != nil {
```
