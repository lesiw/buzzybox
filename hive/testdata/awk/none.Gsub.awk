BEGIN {
    s = "Hello, world!"
    n = gsub("world", "awk", s)
    print s, n

    s = "foo bar baz"
    n = gsub(/foo/, "biz", s)
    print s, n

    s = "foo bar baz"
    n = gsub(/foo/, "||&||", s)
    print s, n

    s = "foo bar baz"
    n = gsub(/foo/, "&buzz", s)
    print s, n

    s = "foo bar baz"
    n = gsub(/foo/, "||\&||", s)
    print s, n

    s = "foo bar baz"
    n = gsub(/foo/, "||\\&||", s)
    print s, n

    s = "foo foo foo"
    n = gsub(/foo/, "biz", s)
    print s, n

    s = "foo bar baz"
    n = gsub(/biz/, "foo", s)
    print s, n
}
