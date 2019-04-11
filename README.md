
About
-----

Simple encrypted storage for named texts

Installation
------------

`go get -u github.com/AnimusPEXUS/tpwsm2`
`go install github.com/AnimusPEXUS/tpwsm2`

Usage
-----

```
cd dir_where_to_store

> tpwsm2
(now enter new password)

> !help
  !h, !help     - help

  !l            - list
  !d name       - delete
  !n name name2 - rename

  !s            - save
  !r            - change password
  !quit, !exit  - exit (Ctrl+d also)

  other_text    - used as name - start editing existing record.
                  if prefixed with '+' - create if not exists.
```
