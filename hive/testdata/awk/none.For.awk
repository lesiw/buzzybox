BEGIN {
    for (i = 0; i < 5;)
        print i++
    for(;;) {
        print j++
        if (j >= 5)
            break
    }
    print "done"
}
