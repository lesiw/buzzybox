BEGIN {
    fn()
}

function fn() {
    if (0) return
    print "done"
}
