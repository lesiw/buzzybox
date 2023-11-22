BEGIN {
    split("foobarbaz", arr, "")
    for (i = 1; i <= length(arr); i++)
        print i ":", arr[i]
}
