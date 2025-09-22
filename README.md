# go-file-splitter

Go言語のソースファイルを関数単位で分割するCLIツールです。パブリック関数、メソッド、テスト関数を個別のファイルに分割できます。

## 機能

- **パブリック関数の分割**: Goファイル内のパブリック関数（大文字で始まる関数）を個別ファイルに分割
- **パブリックメソッドの分割**: 構造体のパブリックメソッドを分割（2つの戦略から選択可能）
- **テスト関数の分割**: `Test`で始まるテスト関数を個別ファイルに分割
- **定数・変数・型定義の処理**: パブリックな定義を`common.go`にまとめて出力
- **コメントの保持**: ドキュメントコメント、インラインコメント、スタンドアロンコメントを適切に処理
- **インポートの最適化**: 使用されているパッケージのみをインポート

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
- `-method-strategy <strategy>`: メソッドの分割戦略を指定
  - `separate` (デフォルト): 各メソッドを個別ファイルに分割
  - `with-struct`: 構造体と関連メソッドを同一ファイルにまとめる
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

# メソッドを構造体と同じファイルにまとめる戦略で分割
go-file-splitter -method-strategy with-struct ./pkg

# メソッドを個別ファイルに分割（デフォルト動作）
go-file-splitter -method-strategy separate ./pkg
```

## 出力構造

### デフォルト戦略（separate）
各パブリック関数・メソッドが個別のファイルに分割されます：
```
output/
├── common.go              # 定数、変数、型定義
├── function_name.go       # パブリック関数
├── type_name_method.go    # メソッド（個別ファイル）
└── test_function.go       # テスト関数
```

### with-struct戦略
構造体とそのメソッドが同一ファイルにまとめられます：
```
output/
├── common.go              # 定数、変数、その他の型定義
├── function_name.go       # パブリック関数
├── type_name.go           # 構造体とそのすべてのメソッド
└── test_function.go       # テスト関数
```

## 主な改善点

このツールの最近のアップデートには以下が含まれます：

- **テストカバレッジの向上**: 74.8%のカバレッジを達成
- **コード品質の改善**: golangci-lintによるすべてのエラーを解決
- **リファクタリング**: 複雑な関数を分割し、保守性を向上
- **インポートの最適化**: 実際に使用されているパッケージのみをインポート
- **コメント処理の改善**: スタンドアロンコメントとインラインコメントの適切な処理
