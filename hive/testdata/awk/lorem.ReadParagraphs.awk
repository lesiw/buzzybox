BEGIN {
    RS = ""
}
{ print $0, "¶" }
