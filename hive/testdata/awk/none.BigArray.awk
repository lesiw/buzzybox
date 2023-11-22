BEGIN {
    arr[x] = -1
    arr[1,2] = -2
    for (i = 0; i < 300; i++)
        arr[i] = i
    for (k in arr)
        print arr[k]
}
