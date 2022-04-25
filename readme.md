<h3 align=center><b>vi-paths</b></h3>
<p align=center><i>edit files and directories with $EDITOR as if they were a text file</i></p>

---

### installation

```shell
    $ go install go.senan.xyz/vi-paths@latest
```

### usage

```shell
    $ export EDITOR=vi
    $ vi-paths [-dry-run] [file] ...
```

### example

```shell
    $ vi-paths ~/music/albums/The Fall/**
    # to rename/move a file/dir, edit the line
    # to delete a file/dir, clear the line
```

### todo

- [ ] add more safety checks

---

[![asciicast](https://asciinema.org/a/TOtkyLZHceizsfHNlHBdk1Jzs.svg)](https://asciinema.org/a/TOtkyLZHceizsfHNlHBdk1Jzs)
