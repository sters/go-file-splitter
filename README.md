# go-test-file-splitter

A CLI tool that splits Go test files by individual test functions (TestXxxx).

## Usage

```shell
go install github.com/sters/go-test-file-splitter@latest
```

```shell
go-test-file-splitter <directory>
```

### Example

```shell
# Split all test files in the current directory
go-test-file-splitter .

# Split test files in a specific package
go-test-file-splitter ./pkg/mypackage
```
