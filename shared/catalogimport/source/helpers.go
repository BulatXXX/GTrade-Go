package source

func applyLimit(total, limit int) int {
	if limit > 0 && limit < total {
		return limit
	}
	return total
}
