package build

import "fmt"

var Time = "N/A"
var User = "developer"
var Version = "development"

func CompleteVersion() string {
    return fmt.Sprintf("%s built on %s by %s", Version, Time, User)
}
