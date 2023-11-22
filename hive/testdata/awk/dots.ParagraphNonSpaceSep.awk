BEGIN {RS = ""; FS = "."}
{for (i = 1; i <= NF; i++) print "record", NR, "field", i, $i}
