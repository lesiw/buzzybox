BEGIN {
    fn()
}

function fn() {
    print "good"
    return
    print "bad"
}
