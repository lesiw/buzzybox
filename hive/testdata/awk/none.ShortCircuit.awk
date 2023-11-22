BEGIN {
    print 0 && 1/0; print "done"
    print 1 || 1/0; print "done"
}
