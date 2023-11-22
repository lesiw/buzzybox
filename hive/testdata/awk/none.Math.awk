BEGIN {
    print (42 - 2)
    print (42 + 2)
    print (42 / 2)
    print (42 * 2)
    print (2^3)
    print (2**3)
    print -0;
    x = 42; x -= 2; print x
    x = 42; x += 2; print x
    x = 42; x /= 2; print x
    x = 42; x *= 2; print x
    x = 2; x ^= 3; print x
    x = 2; x **= 3; print x
    print .2 + .2
}
