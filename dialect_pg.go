package hood

import (
	"fmt"
	"strings"
)

type DialectPg struct{}

func (d *DialectPg) Name() string {
	return "postgres"
}

func (d *DialectPg) Pk() string {
	return "id"
}

func (d *DialectPg) Quote(s string) string {
	q := `"`
	return strings.Join([]string{q, s, q}, "")
}

func (d *DialectPg) Marker(pos int) string {
	return fmt.Sprintf("$%d", pos)
}
