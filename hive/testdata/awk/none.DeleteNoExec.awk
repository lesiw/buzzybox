BEGIN {
    arr[1] = "foo"
    arr[2] = "bar"
    arr[3] = "baz"
    if (0)
        delete arr[2]
    print arr[1], arr[2], arr[3]
}
