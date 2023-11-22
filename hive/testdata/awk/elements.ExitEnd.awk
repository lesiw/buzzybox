END { print "good1" }
END {
    print "good2"
    exit 0
    print "bad1"
}
END { print "bad2" }
