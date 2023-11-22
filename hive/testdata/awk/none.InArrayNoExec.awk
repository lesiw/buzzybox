BEGIN {
    if (0) {
        a["foo"] = "bar"
        print ("foo" in a)
        print ("baz" in a)

        b["foo", "bar"] = "baz"
        print (("foo", "bar") in b)
    }
    print "done"
}
