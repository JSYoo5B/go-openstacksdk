// Package tokens creates, reads, validates and revokes Keystone tokens.
// ValidateToken, CheckToken and RevokeToken act on a Python-style subject
// token: ValidateToken returns the raw token record, and CheckToken reports
// true only for a 200 HEAD.
package tokens
