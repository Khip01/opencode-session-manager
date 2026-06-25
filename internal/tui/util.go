package tui

func statusSummary(orphans, active int) string {
	if orphans == 0 && active == 0 {
		return "no sessions found"
	}
	if orphans == 0 {
		return formatCount(active, "active")
	}
	return formatCount(orphans, "orphan") + ", " + formatCount(active, "active")
}

func formatCount(n int, label string) string {
	if n == 1 {
		return "1 " + label
	}
	return itoa(n) + " " + label + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
