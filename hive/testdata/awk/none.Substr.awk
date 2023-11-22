BEGIN {
    print substr("foo", 1, 10)
    print substr("foobar", 4)
    print substr("foo", -1)
    print substr("foobarbaz", 4, 3)
    print substr("foobarbaz", 7, 3)
    print substr("foo", 0, -5)
    print substr("", 2, 1)
    print substr("x", 1, 1)
    print substr("x", 2, 1)
    print substr("x", 2, 2)
}
