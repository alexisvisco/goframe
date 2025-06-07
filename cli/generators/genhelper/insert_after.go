package genhelper

func InsertLineAfter(lines []string, target string, newLine string) []string {
	for i, line := range lines {
		if line == target {
			lines = append(lines[:i+1], lines[i:]...) // Shift elements to the right
			lines[i+1] = newLine                      // Insert new line after target
			return lines
		}
	}
	return lines // Return original if target not found
}
