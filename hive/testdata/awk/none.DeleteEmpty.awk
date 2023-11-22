BEGIN {
    arr["foo"] = "bar"
    delete arr["foo"]
    i = 0
    for (x in arr) i++
    print i
}
