BEGIN {RS = ""}
{for (i = 1; i <= NF; i++) print "record", NR, "field", i, $i}
