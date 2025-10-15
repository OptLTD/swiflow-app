package errors

import "fmt"

var ErrorQuery = fmt.Errorf("mysql query error")
var ErrorConfig = fmt.Errorf("mysql config error")
var ErrorConnect = fmt.Errorf("mysql connect error")
