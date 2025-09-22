# go-file-splitter

Go言語のソースファイルを関数単位で分割するCLIツールです。パブリック関数やテスト関数を個別のファイルに分割できます。

## 機能

- **パブリック関数の分割**: Goファイル内のパブリック関数（大文字で始まる関数）を個別ファイルに分割
- **テスト関数の分割**: `Test`で始まるテスト関数を個別ファイルに分割
- コメントや構造体、型定義も適切に処理

## インストール

```shell
go install github.com/sters/go-file-splitter@latest
```

## 使い方

```shell
go-file-splitter [options] <directory>
```

### オプション

- `-public-func` (デフォルト: true): パブリック関数を個別ファイルに分割
- `-test-only`: テスト関数のみを分割（`-public-func`を上書き）
- `-version`: バージョン情報を表示

### 例

```shell
# カレントディレクトリ内のGoファイルのパブリック関数を分割（デフォルト動作）
go-file-splitter .

# 特定ディレクトリのパブリック関数を分割
go-file-splitter ./pkg/mypackage

# テスト関数のみを分割
go-file-splitter -test-only ./test

# パブリック関数を明示的に分割
go-file-splitter -public-func ./src
```
