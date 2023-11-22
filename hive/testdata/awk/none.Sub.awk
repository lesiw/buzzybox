BEGIN {
    s = "Hello, world!"
    n = sub("world", "awk", s)
    print s, n

    s = "foo bar baz"
    n = sub(/foo/, "biz", s)
    print s, n

    s = "foo bar baz"
    n = sub(/foo/, "||&||", s)
    print s, n

    s = "foo bar baz"
    n = sub(/foo/, "&buzz", s)
    print s, n

    s = "foo bar baz"
    n = sub(/foo/, "||\&||", s)
    print s, n

    s = "foo bar baz"
    n = sub(/foo/, "||\\&||", s)
    print s, n

    s = "foo foo foo"
    n = sub(/foo/, "biz", s)
    print s, n

    s = "foo bar baz"
    n = sub(/biz/, "foo", s)
    print s, n
}
