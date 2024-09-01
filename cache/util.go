package cache

// func getFileWithExtension(file string) string {
// 	// find the file matching pattern with the longest name
// 	matches, err := filepath.Glob(file + "*")
// 	util.CheckPanicOld(err)
// 	for _, match := range matches {
// 		if len(match) > len(file) {
// 			file = match
// 		}
// 	}
// 	return file
// }

// func formatRawFileElement(file string) string {
// 	bytes, size, _ := uil.isBinaryFile(file)
// 	s := string(bytes)
// 	html := "<xmp>" + s + "\n\n"
// 	if size > maxLength {
// 		html += "[...]"
// 	}
// 	html += "</xmp>"
// 	return html
// }

// func formatRawStringElement(raw string) string {
// 	size := len(raw)
// 	bytes := raw[0:util.Min(size, byteLength)]
// 	s := string(bytes)
// 	html := "<xmp>" + s + "\n\n"
// 	if size > byteLength {
// 		html += "[...]"
// 	}
// 	html += "</xmp>"
// 	return html
// }
