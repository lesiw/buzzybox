BEGIN {
    while ("cat testdata/awk/elements" | getline foo) {
        print "getline:", foo
    }
}
