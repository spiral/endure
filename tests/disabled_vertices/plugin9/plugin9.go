package plugin9

import "github.com/spiral/endure/tests/disabled_vertices/plugin6"

type Plugin9 struct {
}

func (p9 *Plugin9) Init(plugin6 plugin6.Plugin6) error {
	return nil
}
