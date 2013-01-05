package hood

type Migration interface {
	// Up executes the commands on apply.
	Up(hood *Hood)

	// Down executes the commands on rollback.
	Down(hood *Hood)
}
