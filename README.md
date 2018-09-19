
About
-----

Simple encrypted storage for named texts

Installation
------------

`go get -u github.com/AnimusPEXUS/tpwsm`
`go install github.com/AnimusPEXUS/tpwsm`

Usage
-----

```
cd dir_where_to_store

> tpwsm
(now enter new password)

> !help
  !h, !help    - help

  !l           - list
  !d id        - delete
  !n id name   - rename

  !r           - change password
  !quit, !exit - exit (Ctrl+d also)

  other_text   - used as name - start editing existing record.
                 if prefixed with '+' - create if not exists.
```
