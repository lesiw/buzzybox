BEGIN {
    start = match("Hello 世界", / .+/)
    print start, RSTART, RLENGTH

    start = match("Hello 世界", /界/)
    print start, RSTART, RLENGTH
}
