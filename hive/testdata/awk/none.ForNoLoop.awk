BEGIN {
    for (i = 0;i > 100;i++) {
        print "bad"
    }
    a["foo"] = "bar"
    delete a["foo"]
    for (x in a) {
        print "bad"
    }
    print "done"
}
