BEGIN {
    print hello("world")
}

function hello(target) {
    return "Hello, " target
}
