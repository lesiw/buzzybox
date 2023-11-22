BEGIN {
    start = match("foo bar baz", /b?r/)
    print start, RSTART, RLENGTH

    start = match("foo bar baz", /x/)
    print start, RSTART, RLENGTH
}
