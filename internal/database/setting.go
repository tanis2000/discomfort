package database

type Setting struct {
    ID                  int
    Version             int
    MigratedFromImageDB bool
}
