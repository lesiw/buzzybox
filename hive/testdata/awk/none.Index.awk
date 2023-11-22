BEGIN {
    print index("foobarbaz", "bar")
    print index("foofoofoo", "bar")
    print index("barbarbar", "bar")
    print index("bar", "foobar")
    print index("foo", "")
    print index("", "foo")
    print index("", "")
}
