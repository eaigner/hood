package hood

import (
	"fmt"
)

type DialectPg struct{}

func (d *DialectPg) Pk() string {
	return "id"
}

func (d *DialectPg) Quote() rune {
	return '"'
}

func (d *DialectPg) Marker(pos int) string {
	return fmt.Sprintf("$%d", pos)
}
