package usecases

import "github.com/google/uuid"

// goRunner is the production AsyncRunner that launches goroutines.
type goRunner struct{}

func NewGoRunner() *goRunner { return &goRunner{} }

func (r *goRunner) Go(fn func()) { go fn() }

// uuidGenerator is the production IDGenerator using google/uuid.
type uuidGenerator struct{}

func NewUUIDGenerator() *uuidGenerator { return &uuidGenerator{} }

func (g *uuidGenerator) NewID() string { return uuid.New().String() }
