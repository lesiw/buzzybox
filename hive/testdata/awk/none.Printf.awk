BEGIN {
    printf("print %%s %s\n", "2")
    printf("print %%s %s\n", 3)
    printf("print %%s %s\n", "")

    printf("print %%d %d\n", 1)
    printf("print %%d %d\n", "4")
    printf("print %%d %d\n", "")

    printf("print %%u %u\n", 1)
    printf("print %%u %u\n", -1)
    printf("print %%u %u\n", "")

    printf("print %%c %c\n", 42)
    printf("print %%c %c\n", "42")
    printf("print %%c %c\n", 227)
    printf("print %%c %c\n", "0abc")

    printf("print %%o %o\n", "2")
    printf("print %%o %o\n", 3)
    printf("print %%o %o\n", -1)
    printf("print %%o %o\n", "")

    printf("print %%x %x\n", "2")
    printf("print %%x %x\n", 3)
    printf("print %%x %x\n", -1)
    printf("print %%x %x\n", "")

    printf("print %%X %X\n", "2")
    printf("print %%X %X\n", 3)
    printf("print %%X %X\n", -1)
    printf("print %%X %X\n", "")
}
