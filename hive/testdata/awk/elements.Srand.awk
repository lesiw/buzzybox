BEGIN	{ k = 3; n = 10; s = srand(42); print s }
{	if (n <= 0) exit
	if (rand() <= k/n) { print; k-- }
	n--
}
