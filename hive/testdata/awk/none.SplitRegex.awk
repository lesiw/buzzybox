BEGIN {
    n = split("foo.bar.baz", arr, /./)
    print n
    for (i = 1; i <= length(arr); i++)
        print i ":", arr[i]
}
