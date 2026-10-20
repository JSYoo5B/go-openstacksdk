package rest

// HeaderNextLinks returns next targets in header order using the shared quoted
// HTTP Link parser. The caller owns precedence and selection across headers.
func HeaderNextLinks(header string) ([]string, error) {
	return headerNextLinks(header)
}
