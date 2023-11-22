BEGIN {RS = ""; FS = "[0-9]+" }
{for (i = 1; i <= NF; i++) print "record", NR, "field", i, $i}
