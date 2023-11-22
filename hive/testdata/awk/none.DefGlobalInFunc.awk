BEGIN {
    fn1()
}

function fn1() {
    x = 42
    fn2()
    print x
}

function fn2() {
    x++
}
