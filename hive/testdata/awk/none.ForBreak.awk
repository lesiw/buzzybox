BEGIN {
    for (i = 0;;i++) {
        if (i > 5) {
            break
        }
        print i
    }
    a["one"] = "foo"
    a["two"] = "bar"
    a["three"] = "baz"
    for (x in a) {
        print a[x]
        break
    }
}
