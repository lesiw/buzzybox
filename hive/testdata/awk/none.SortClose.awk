BEGIN {
    print "B" | "sort"
    print "A" | "sort"
    print "C" | "sort"
    close("sort")
    print "end of program"
}
