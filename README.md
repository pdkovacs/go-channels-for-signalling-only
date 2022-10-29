# go-channels-for-signalling-only

`internal/strsli_to_strsli.go` has been prepared for this coding challange on Twitter: https://twitter.com/teivah/status/1584486051434749953 .

Test it with something like

```
    i=0; while go clean -testcache && go test -v -timeout 10s ./... -testify.m TestWithOneInput; do echo $((i++)); done
```

. Stop with Ctrl+C whenever you feel confident enough about the number of PASSes.
