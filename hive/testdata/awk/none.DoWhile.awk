BEGIN {
    do {
        print "loop1"
    } while(0)
    i = 5
    do {
        print i--
    } while(i > 0)
    do {
        print "loop2"
        break
    } while(true)
}
