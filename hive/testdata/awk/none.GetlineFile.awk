BEGIN {
    print "foo bar baz" > "tmp"
    while(getline < "tmp")
        print $3
}
