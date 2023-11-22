BEGIN {
    while ("cat testdata/awk/elements" | getline) {
        print "getline:", $2, $4
    }
}
