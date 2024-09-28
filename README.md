# gostat

[English](README_EN.md)

gostat 是一个基于 Go 语言的系统监控工具，能够在终端界面中提供实时的网络统计信息。

目前仅支持Linux系统。

## 特性

- 实时获取系统的基础信息
- 基于终端的用户界面（使用 tview）
- 轻量级且易于使用

## 安装

使用以下命令安装 gostat：

```bash
curl -sSL https://raw.githubusercontent.com/Merbeleue/gostat/main/install.sh | bash
 ```

## 使用方法

安装后，您可以直接在命令行中使用 gostat：

```bash
gostat
```

## 依赖项

本项目依赖于以下优秀的库：

- [tview](https://github.com/rivo/tview)：用于创建丰富的终端用户界面
- [tcell](https://github.com/gdamore/tcell)：用于底层终端处理

我们向这些项目的维护者和贡献者表示诚挚的感谢。

## 贡献

欢迎贡献！请随时提交 Pull Request。

## 致谢

- 感谢所有项目的贡献者和用户
- 特别感谢 Go 社区提供的优秀工具和库
