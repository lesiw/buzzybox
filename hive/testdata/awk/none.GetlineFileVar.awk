BEGIN {
    print "foo bar baz" > "tmp"
    while(getline line < "tmp")
        print "line:", line
}
