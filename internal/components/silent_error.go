package components

import "errors"

// ErrSilent signals a failure the command already reported to the user;
// bootstrap exits 1 without printing anything else.
var ErrSilent = errors.New("failure already reported")
