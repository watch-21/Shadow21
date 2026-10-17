package anonsurf

import "gitlab.com/parrotsec/packages/anonsurf/internal/tor"

func (c Controller) generateTorrc() error {
	return (tor.Configurator{}).Generate()
}
