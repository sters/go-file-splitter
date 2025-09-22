# go-file-splitter

A CLI tool that splits Go source files into individual files by function. It can separate public functions, methods, and test functions into individual files.

## Features

- **Public Function Splitting**: Splits public functions (starting with uppercase) into individual files
- **Public Method Splitting**: Splits struct public methods (with two strategies to choose from)
- **Test Function Splitting**: Splits test functions starting with `Test` into individual files
- **Constants, Variables, and Type Definitions**: Groups public definitions into `common.go`
- **Comment Preservation**: Properly handles doc comments, inline comments, and standalone comments
- **Import Optimization**: Only imports packages that are actually used

## Installation

```shell
go install github.com/sters/go-file-splitter@latest
```

## Usage

```shell
go-file-splitter [options] <directory>
```

### Options

- `-public-func` (default: true): Split public functions into individual files
- `-test-only`: Split only test functions (overrides `-public-func`)
- `-method-strategy <strategy>`: Specify method splitting strategy
  - `separate` (default): Split each method into individual files
  - `with-struct`: Group struct and its methods in the same file
- `-version`: Show version information

### Examples

```shell
# Split public functions in current directory (default behavior)
go-file-splitter .

# Split public functions in specific directory
go-file-splitter ./pkg/mypackage

# Split only test functions
go-file-splitter -test-only ./test

# Explicitly split public functions
go-file-splitter -public-func ./src

# Split with methods grouped with their structs
go-file-splitter -method-strategy with-struct ./pkg

# Split methods into individual files (default behavior)
go-file-splitter -method-strategy separate ./pkg
```

## Output Structure

### Default Strategy (separate)
Each public function and method is split into individual files:
```
output/
├── common.go              # Constants, variables, type definitions
├── function_name.go       # Public functions
├── type_name_method.go    # Methods (individual files)
└── test_function.go       # Test functions
```

### with-struct Strategy
Structs and their methods are grouped in the same file:
```
output/
├── common.go              # Constants, variables, other type definitions
├── function_name.go       # Public functions
├── type_name.go           # Struct with all its methods
└── test_function.go       # Test functions
```

## Recent Improvements

Recent updates to this tool include:

- **Improved Test Coverage**: Achieved 74.8% coverage
- **Code Quality Improvements**: Resolved all golangci-lint errors
- **Refactoring**: Split complex functions to improve maintainability
- **Import Optimization**: Only imports packages that are actually used
- **Enhanced Comment Handling**: Properly handles standalone and inline comments
